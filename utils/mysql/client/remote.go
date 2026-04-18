package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/files"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

type Remote struct {
	Type     string
	Client   *sql.DB
	Database string
	User     string
	Password string
	Address  string
	Port     uint

	SSL        bool
	RootCert   string
	ClientKey  string
	ClientCert string
	SkipVerify bool
}

func NewRemote(db Remote) *Remote {
	return &db
}

func (r *Remote) Create(info CreateInfo) error {
	createSql := fmt.Sprintf("create database `%s` default character set %s collate %s", info.Name, info.Format, formatMap[info.Format])
	if err := r.ExecSQL(createSql, info.Timeout); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "error 1007") {
			return errors.New(constant.ErrDatabaseIsExist)
		}
		return err
	}

	if err := r.CreateUser(info, true); err != nil {
		_ = r.ExecSQL(fmt.Sprintf("drop database if exists `%s`", info.Name), info.Timeout)
		return err
	}

	return nil
}

func (r *Remote) CreateUser(info CreateInfo, withDeleteDB bool) error {
	var userlist []string
	if strings.Contains(info.Permission, ",") {
		ips := strings.Split(info.Permission, ",")
		for _, ip := range ips {
			if len(ip) != 0 {
				userlist = append(userlist, fmt.Sprintf("'%s'@'%s'", info.Username, ip))
			}
		}
	} else {
		userlist = append(userlist, fmt.Sprintf("'%s'@'%s'", info.Username, info.Permission))
	}

	for _, user := range userlist {
		if err := r.ExecSQL(fmt.Sprintf("create user %s identified by '%s';", user, info.Password), info.Timeout); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "error 1396") {
				return errors.New(constant.ErrUserIsExist)
			}
			if withDeleteDB {
				_ = r.Delete(DeleteInfo{
					Name:        info.Name,
					Version:     info.Version,
					Username:    info.Username,
					Permission:  info.Permission,
					ForceDelete: true,
					Timeout:     300})
			}
			return err
		}
		grantStr := fmt.Sprintf("grant all privileges on `%s`.* to %s", info.Name, user)
		if info.Name == "*" {
			grantStr = fmt.Sprintf("grant all privileges on *.* to %s", user)
		}
		if strings.HasPrefix(info.Version, "5.7") || strings.HasPrefix(info.Version, "5.6") {
			grantStr = fmt.Sprintf("%s identified by '%s' with grant option;", grantStr, info.Password)
		} else {
			grantStr = grantStr + " with grant option;"
		}
		if err := r.ExecSQL(grantStr, info.Timeout); err != nil {
			if withDeleteDB {
				_ = r.Delete(DeleteInfo{
					Name:        info.Name,
					Version:     info.Version,
					Username:    info.Username,
					Permission:  info.Permission,
					ForceDelete: true,
					Timeout:     300})
			}
			return err
		}
	}
	return nil
}

func (r *Remote) Delete(info DeleteInfo) error {
	var userlist []string
	if strings.Contains(info.Permission, ",") {
		ips := strings.Split(info.Permission, ",")
		for _, ip := range ips {
			if len(ip) != 0 {
				userlist = append(userlist, fmt.Sprintf("'%s'@'%s'", info.Username, ip))
			}
		}
	} else {
		userlist = append(userlist, fmt.Sprintf("'%s'@'%s'", info.Username, info.Permission))
	}

	for _, user := range userlist {
		if strings.HasPrefix(info.Version, "5.6") {
			if err := r.ExecSQL(fmt.Sprintf("drop user %s", user), info.Timeout); err != nil && !info.ForceDelete {
				return err
			}
		} else {
			if err := r.ExecSQL(fmt.Sprintf("drop user if exists %s", user), info.Timeout); err != nil && !info.ForceDelete {
				return err
			}
		}
	}
	if len(info.Name) != 0 {
		if err := r.ExecSQL(fmt.Sprintf("drop database if exists `%s`", info.Name), info.Timeout); err != nil && !info.ForceDelete {
			return err
		}
	}
	if !info.ForceDelete {
		global.LOG.Info("execute delete database sql successful, now start to drop uploads and records")
	}

	return nil
}

func (r *Remote) ChangePassword(info PasswordChangeInfo) error {
	if info.Username != "root" {
		var userlist []string
		if strings.Contains(info.Permission, ",") {
			ips := strings.Split(info.Permission, ",")
			for _, ip := range ips {
				if len(ip) != 0 {
					userlist = append(userlist, fmt.Sprintf("'%s'@'%s'", info.Username, ip))
				}
			}
		} else {
			userlist = append(userlist, fmt.Sprintf("'%s'@'%s'", info.Username, info.Permission))
		}

		for _, user := range userlist {
			passwordChangeSql := fmt.Sprintf("set password for %s = password('%s')", user, info.Password)
			if !strings.HasPrefix(info.Version, "5.7") && !strings.HasPrefix(info.Version, "5.6") {
				passwordChangeSql = fmt.Sprintf("ALTER USER %s IDENTIFIED BY '%s';", user, info.Password)
			}
			if err := r.ExecSQL(passwordChangeSql, info.Timeout); err != nil {
				return err
			}
		}
		return nil
	}

	hosts, err := r.ExecSQLForHosts(info.Timeout)
	if err != nil {
		return err
	}
	for _, host := range hosts {
		if host == "%" || host == "localhost" {
			passwordRootChangeCMD := fmt.Sprintf("set password for 'root'@'%s' = password('%s')", host, info.Password)
			if !strings.HasPrefix(info.Version, "5.7") && !strings.HasPrefix(info.Version, "5.6") {
				passwordRootChangeCMD = fmt.Sprintf("alter user 'root'@'%s' identified by '%s';", host, info.Password)
			}
			if err := r.ExecSQL(passwordRootChangeCMD, info.Timeout); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Remote) ChangeAccess(info AccessChangeInfo) error {
	if info.Username == "root" {
		info.OldPermission = "%"
		info.Name = "*"
		info.Password = r.Password
	}
	if info.Permission != info.OldPermission {
		if err := r.Delete(DeleteInfo{
			Version:     info.Version,
			Username:    info.Username,
			Permission:  info.OldPermission,
			ForceDelete: true,
			Timeout:     300}); err != nil {
			return err
		}
		if info.Username == "root" {
			return nil
		}
	}
	if err := r.CreateUser(CreateInfo{
		Name:       info.Name,
		Version:    info.Version,
		Username:   info.Username,
		Password:   info.Password,
		Permission: info.Permission,
		Timeout:    info.Timeout,
	}, false); err != nil {
		return err
	}
	if err := r.ExecSQL("flush privileges", 300); err != nil {
		return err
	}
	return nil
}

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

	// 如果 info.Format 看起来像文件名（包含点），不要把它当 charset 使用
	charset := ""
	if info.Format != "" && !strings.Contains(info.Format, ".") {
		charset = info.Format
	} else if info.Format != "" {
		global.LOG.Warnf("ignoring invalid charset value: %s", info.Format)
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
	if charset != "" {
		mysqldumpArgs = append(mysqldumpArgs, "--default-character-set="+charset)
	}
	if info.Name != "" {
		mysqldumpArgs = append(mysqldumpArgs, info.Name)
	}

	cmdArgs := append(dockerArgs, mysqldumpArgs...)
	global.LOG.Debugf("docker args: %v", cmdArgs) // 不要打印密码（我们已经通过 env 传递）

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(info.Timeout)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
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
	// 打开备份文件
	fi, err := os.Open(info.SourceFile)
	if err != nil {
		return err
	}
	defer fi.Close()

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
	// 仅在 info.Format 看起来像 charset 时添加
	if info.Format != "" && !strings.Contains(info.Format, ".") {
		dockerArgs = append(dockerArgs, "--default-character-set="+info.Format)
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
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)

	// 若为 gzip 文件，解压后作为 stdin；否则直接把文件作为 stdin
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

func ensureDockerImage(image string, policy string, timeout uint) error {
	policy = strings.TrimSpace(strings.ToLower(policy))
	if policy == "" {
		policy = "missing"
	}
	if policy != "missing" && policy != "always" && policy != "never" {
		policy = "missing"
	}

	exists := dockerImageExists(image)
	if exists && policy != "always" {
		return nil
	}
	if !exists && policy == "never" {
		return fmt.Errorf("docker image not found locally: %s", image)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "pull", image)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return errors.New(constant.ErrExecTimeOut)
		}
		return fmt.Errorf("docker pull %s failed: %s", image, strings.TrimSpace(string(out)))
	}
	return nil
}

func dockerImageExists(image string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", image)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (r *Remote) SyncDB(version string) ([]SyncDBInfo, error) {
	var datas []SyncDBInfo
	rows, err := r.Client.Query("select schema_name, default_character_set_name from information_schema.SCHEMATA")
	if err != nil {
		return datas, err
	}
	defer rows.Close()

	for rows.Next() {
		var dbName, charsetName string
		if err = rows.Scan(&dbName, &charsetName); err != nil {
			return datas, err
		}
		if dbName == "information_schema" || dbName == "mysql" || dbName == "performance_schema" || dbName == "sys" || dbName == "__recycle_bin__" || dbName == "recycle_bin" {
			continue
		}
		dataItem := SyncDBInfo{
			Name:      dbName,
			From:      "remote",
			MysqlName: r.Database,
			Format:    charsetName,
		}
		userRows, err := r.Client.Query("select user,host from mysql.db where db = ?", dbName)
		if err != nil {
			global.LOG.Debugf("sync user of db %s failed, err: %v", dbName, err)
			dataItem.Permission = "%"
			datas = append(datas, dataItem)
			continue
		}

		var permissionItem []string
		isLocal := true
		i := 0
		for userRows.Next() {
			var user, host string
			if err = userRows.Scan(&user, &host); err != nil {
				return datas, err
			}
			if user == "root" {
				continue
			}
			if i == 0 {
				dataItem.Username = user
			}
			if dataItem.Username == user && host == "%" {
				isLocal = false
				dataItem.Permission = "%"
			} else if dataItem.Username == user && host != "localhost" {
				isLocal = false
				permissionItem = append(permissionItem, host)
			}
			i++
		}
		if len(dataItem.Username) == 0 {
			dataItem.Permission = "%"
		} else {
			if isLocal {
				dataItem.Permission = "localhost"
			}
			if len(dataItem.Permission) == 0 {
				dataItem.Permission = strings.Join(permissionItem, ",")
			}
		}
		datas = append(datas, dataItem)
	}
	if err = rows.Err(); err != nil {
		return datas, err
	}
	return datas, nil
}

func (r *Remote) Close() {
	_ = r.Client.Close()
}

func (r *Remote) ExecSQL(command string, timeout uint) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	if _, err := r.Client.ExecContext(ctx, command); err != nil {
		return err
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errors.New(constant.ErrExecTimeOut)
	}

	return nil
}

func (r *Remote) ExecSQLForHosts(timeout uint) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	results, err := r.Client.QueryContext(ctx, "select host from mysql.user where user='root';")
	if err != nil {
		return nil, err
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, errors.New(constant.ErrExecTimeOut)
	}
	var rows []string
	for results.Next() {
		var host string
		if err := results.Scan(&host); err != nil {
			continue
		}
		rows = append(rows, host)
	}
	return rows, nil
}

func loadImage(dbType, version string) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}
	images, err := cli.ImageList(context.Background(), image.ListOptions{})
	if err != nil {
		return "", err
	}

	var candidates []string
	for _, image := range images {
		for _, tag := range image.RepoTags {
			tag = strings.TrimSpace(tag)
			if tag == "" || tag == "<none>:<none>" {
				continue
			}
			if !strings.HasPrefix(tag, dbType+":") {
				continue
			}
			candidates = append(candidates, tag)
		}
	}

	if version == "" {
		if best, ok := pickBestTag(candidates); ok {
			return best, nil
		}
		return loadVersion(dbType, version), nil
	}

	for _, tag := range candidates {
		if dbType == "mariadb" {
			return tag, nil
		}
		if strings.HasPrefix(version, "5.6") && strings.HasPrefix(tag, "mysql:5.6") {
			return tag, nil
		}
		if strings.HasPrefix(version, "5.7") && strings.HasPrefix(tag, "mysql:5.7") {
			return tag, nil
		}
		if strings.HasPrefix(version, "8.") && strings.HasPrefix(tag, "mysql:8.") {
			return tag, nil
		}
	}

	if best, ok := pickBestTag(candidates); ok {
		return best, nil
	}
	return loadVersion(dbType, version), nil
}

func loadVersion(dbType string, version string) string {
	if dbType == "mariadb" {
		return "mariadb:11.3.2"
	}
	if strings.HasPrefix(version, "5.6") {
		return "mysql:5.6.51"
	}
	if strings.HasPrefix(version, "5.7") {
		return "mysql:5.7.44"
	}
	return "mysql:8.2.0"
}

func pickBestTag(tags []string) (string, bool) {
	if len(tags) == 0 {
		return "", false
	}
	best := tags[0]
	for _, t := range tags[1:] {
		if compareDockerTagVersion(t, best) > 0 {
			best = t
		}
	}
	return best, true
}

func compareDockerTagVersion(a, b string) int {
	av := parseTagVersion(a)
	bv := parseTagVersion(b)
	n := len(av)
	if len(bv) > n {
		n = len(bv)
	}
	for i := 0; i < n; i++ {
		ai := 0
		if i < len(av) {
			ai = av[i]
		}
		bi := 0
		if i < len(bv) {
			bi = bv[i]
		}
		if ai != bi {
			if ai > bi {
				return 1
			}
			return -1
		}
	}
	return 0
}

func parseTagVersion(tag string) []int {
	parts := strings.SplitN(tag, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	v := parts[1]
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	raw := strings.Split(v, ".")
	out := make([]int, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return out
		}
		out = append(out, n)
	}
	return out
}

func sslSkip(version, dbType string) string {
	if dbType == constant.AppMariaDB || strings.HasPrefix(version, "5.6") || strings.HasPrefix(version, "5.7") {
		return "--skip-ssl"
	}
	return "--ssl-mode=DISABLED"
}
