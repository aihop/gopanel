package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/files"
)

func (r *Remote) Backup(info BackupInfo) error {
	fileOp := files.NewFileOp()
	if !fileOp.Stat(info.TargetDir) {
		if err := os.MkdirAll(info.TargetDir, os.ModePerm); err != nil {
			return fmt.Errorf("mkdir %s failed, err: %v", info.TargetDir, err)
		}
	}
	outPath := path.Join(info.TargetDir, info.FileName)
	outfile, err := os.OpenFile(outPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open file %s failed, err: %v", outPath, err)
	}
	defer outfile.Close()

	dumpCmd := "mysqldump"
	if r.Type == constant.AppMariaDB {
		dumpCmd = "mariadb-dump"
	}

	hostDumpCmd := dumpCmd
	if hostDumpCmd, err = ensureHostDumpCmd(dumpCmd); err != nil {
		return err
	} else if hostDumpCmd != "" {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(info.Timeout)*time.Second)
		defer cancel()
		if err := r.backupWithHostClient(ctx, info, hostDumpCmd, outfile); err == nil {
			return nil
		}
	}

	// 构造 docker run 参数，使用 MYSQL_PWD 环境变量传密码，避免 -pPASSWORD 警告
	dockerArgs := []string{
		"run", "--rm", "--net=host", "-i",
		"-e", "MYSQL_PWD=" + r.Password,
	}
	// image
	image, err := loadImage(info.Type, info.Version)
	if err != nil {
		return err
	}

	policy := strings.ToLower(strings.TrimSpace(os.Getenv("GOPANEL_DOCKER_PULL")))
	if policy == "" {
		policy = "missing"
	}
	if err := ensureDockerImage(image, policy, uint(maxInt(int(info.Timeout), 600))); err != nil {
		return err
	}
	dockerArgs = append(dockerArgs, image)

	// 构造 mysqldump 参数（放在 docker run 后）
	mysqldumpArgs := []string{
		dumpCmd,
		"--routines",
		"--single-transaction",
		"--skip-lock-tables",
		"-h", r.Address,
		"-P", fmt.Sprintf("%d", r.Port),
		"-u", r.User,
	}
	// ssl/compat 参数
	if s := sslSkip(info.Version, r.Type); s != "" {
		mysqldumpArgs = append(mysqldumpArgs, s)
	}
	if info.Name != "" {
		mysqldumpArgs = append(mysqldumpArgs, info.Name)
	}

	cmdArgs := append(dockerArgs, mysqldumpArgs...)
	global.LOG.Debugf("docker args: %v", cmdArgs) // 不要打印密码（我们已经通过 env 传递）

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(info.Timeout)*time.Second)
	defer cancel()
	cmd, err := runtimeCommandForDBTool(ctx, cmdArgs...)
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	dumpOut, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe failed: %v", err)
	}

	gzipCmd := exec.Command("gzip", "-cf")
	gzipCmd.Stdin = dumpOut
	gzipCmd.Stdout = outfile

	if err := gzipCmd.Start(); err != nil {
		return fmt.Errorf("start gzip failed: %v", err)
	}

	if err := cmd.Start(); err != nil {
		_ = gzipCmd.Process.Kill()
		return fmt.Errorf("start docker/mysqldump failed: %v, stderr: %s", err, stderr.String())
	}

	if err := cmd.Wait(); err != nil {
		_ = gzipCmd.Process.Kill()
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.New(constant.ErrExecTimeOut)
		}
		return fmt.Errorf("handle backup database failed, err: %v", stderr.String())
	}

	if err := gzipCmd.Wait(); err != nil {
		return fmt.Errorf("gzip failed: %v", err)
	}

	return nil
}

func (r *Remote) Recover(info RecoverInfo) error {
	input, err := openRecoverStream(info.SourceFile, info.Progress)
	if err != nil {
		return err
	}
	defer input.Close()

	// 选择 image
	image, err := loadImage(info.Type, info.Version)
	if err != nil {
		return err
	}

	policy := strings.ToLower(strings.TrimSpace(os.Getenv("GOPANEL_DOCKER_PULL")))
	if policy == "" {
		policy = "missing"
	}
	if err := ensureDockerImage(image, policy, uint(maxInt(int(info.Timeout), 600))); err != nil {
		return err
	}

	// 选择客户端命令: mysql 或 mariadb
	clientCmd := "mysql"
	if r.Type == constant.AppMariaDB {
		clientCmd = "mariadb"
	}

	hostClientCmd := clientCmd
	if hostClientCmd, err = ensureHostMysqlCmd(clientCmd); err != nil {
		return err
	} else if hostClientCmd != "" {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(info.Timeout)*time.Second)
		defer cancel()
		if err := r.recoverWithHostClient(ctx, info, hostClientCmd); err == nil {
			return nil
		}
	}

	// 构造 docker run args，使用 MYSQL_PWD 环境变量传密码
	dockerArgs := []string{
		"run", "--rm", "--net=host", "-i",
		"-e", "MYSQL_PWD=" + r.Password,
		image,
		clientCmd,
		"-h", r.Address,
		"-P", fmt.Sprintf("%d", r.Port),
		"-u", r.User,
	}
	// ssl/兼容参数
	if s := sslSkip(info.Version, r.Type); s != "" {
		// sslSkip 返回以 -- 开头的字符串或空，按需拆分并追加
		parts := strings.Fields(s)
		dockerArgs = append(dockerArgs, parts...)
	}
	// 指定数据库名（可为空，表示从 stdin 执行 SQL）
	if info.Name != "" {
		dockerArgs = append(dockerArgs, info.Name)
	}

	global.LOG.Debugf("docker recover args (password hidden): %v", func() []string {
		safe := make([]string, len(dockerArgs))
		copy(safe, dockerArgs)
		for i := range safe {
			if strings.Contains(safe[i], r.Password) {
				safe[i] = strings.ReplaceAll(safe[i], r.Password, "******")
			}
		}
		return safe
	}())

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(info.Timeout)*time.Second)
	defer cancel()
	cmd, err := runtimeCommandForDBTool(ctx, dockerArgs...)
	if err != nil {
		return err
	}

	cmd.Stdin = input

	// 捕获输出以便返回错误信息
	out, err := cmd.CombinedOutput()
	outStr := strings.ReplaceAll(string(out), "mysql: [Warning] Using a password on the command line interface can be insecure.\n", "")
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.New(constant.ErrExecTimeOut)
		}
		return fmt.Errorf("%s", outStr)
	}
	if strings.HasPrefix(outStr, "ERROR ") || strings.Contains(strings.ToLower(outStr), "error") {
		return fmt.Errorf("%s", outStr)
	}
	return nil
}

func (r *Remote) backupWithHostClient(ctx context.Context, info BackupInfo, dumpCmd string, outfile *os.File) error {
	host := strings.TrimPrefix(r.Address, "[")
	host = strings.TrimSuffix(host, "]")
	args := []string{
		"--routines",
		"--single-transaction",
		"--skip-lock-tables",
		"-h", host,
		"-P", fmt.Sprintf("%d", r.Port),
		"-u", r.User,
	}
	if s := sslSkip(info.Version, r.Type); s != "" {
		parts := strings.Fields(s)
		args = append(args, parts...)
	}
	if info.Name != "" {
		args = append(args, info.Name)
	}
	cmd := exec.CommandContext(ctx, dumpCmd, args...)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+r.Password)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	gzipCmd := exec.CommandContext(ctx, "gzip", "-cf")
	gzipCmd.Stdin = out
	gzipCmd.Stdout = outfile
	var gzErr bytes.Buffer
	gzipCmd.Stderr = &gzErr

	if err := gzipCmd.Start(); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		_ = gzipCmd.Process.Kill()
		return err
	}
	if err := cmd.Wait(); err != nil {
		_ = gzipCmd.Process.Kill()
		return fmt.Errorf("mysqldump failed: %s", strings.TrimSpace(stderr.String()))
	}
	if err := gzipCmd.Wait(); err != nil {
		return fmt.Errorf("gzip failed: %s", strings.TrimSpace(gzErr.String()))
	}
	return nil
}

func (r *Remote) recoverWithHostClient(ctx context.Context, info RecoverInfo, clientCmd string) error {
	host := strings.TrimPrefix(r.Address, "[")
	host = strings.TrimSuffix(host, "]")
	args := []string{
		"-h", host,
		"-P", fmt.Sprintf("%d", r.Port),
		"-u", r.User,
	}
	if s := sslSkip(info.Version, r.Type); s != "" {
		parts := strings.Fields(s)
		args = append(args, parts...)
	}
	if info.Name != "" {
		args = append(args, info.Name)
	}

	cmd := exec.CommandContext(ctx, clientCmd, args...)
	cmd.Env = append(os.Environ(), "MYSQL_PWD="+r.Password)

	fi, err := os.Open(info.SourceFile)
	if err != nil {
		return err
	}
	defer fi.Close()

	if strings.HasSuffix(info.SourceFile, ".gz") {
		gr, err := gzip.NewReader(fi)
		if err != nil {
			return err
		}
		defer gr.Close()
		cmd.Stdin = gr
	} else {
		cmd.Stdin = fi
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		s := strings.TrimSpace(string(out))
		if s == "" {
			return err
		}
		return errors.New(s)
	}
	return nil
}
