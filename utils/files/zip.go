package files

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cmd"
	"github.com/aihop/gopanel/utils/common"
)

type ZipArchiver struct {
}

func NewZipArchiver() ShellArchiver {
	return &ZipArchiver{}
}

func (z ZipArchiver) Extract(filePath, dstDir string, secret string) error {
	if err := checkCmdAvailability("unzip"); err != nil {
		return err
	}
	return cmd.ExecCmd(fmt.Sprintf("unzip -qo %s -d %s", filePath, dstDir))
}

func (z ZipArchiver) Compress(sourcePaths []string, dstFile string, _ string) error {
	var err error
	tmpFile := path.Join(global.CONF.System.TmpDir, fmt.Sprintf("%s%s.zip", common.RandStr(50), time.Now().Format(constant.DateTimeSlimLayout)))
	op := NewFileOp()
	defer func() {
		_ = op.DeleteFile(tmpFile)
		if err != nil {
			_ = op.DeleteFile(dstFile)
		}
	}()
	baseDir := path.Dir(sourcePaths[0])
	relativePaths := make([]string, len(sourcePaths))
	for i, sp := range sourcePaths {
		relativePaths[i] = path.Base(sp)
	}
	cmdStr := fmt.Sprintf("zip -qr %s  %s", tmpFile, strings.Join(relativePaths, " "))
	if err = cmd.ExecCmdWithDir(cmdStr, baseDir); err != nil {
		return err
	}
	if err = op.Mv(tmpFile, dstFile); err != nil {
		return err
	}
	return nil
}

// CompressWithOutput 执行压缩并通过 outputFn 逐行回传输出，使用 -r（非安静模式）展示每个文件
func (z ZipArchiver) CompressWithOutput(sourcePaths []string, dstFile string, _ string, outputFn func(line string)) error {
	var err error
	tmpFile := path.Join(global.CONF.System.TmpDir, fmt.Sprintf("%s%s.zip", common.RandStr(50), time.Now().Format(constant.DateTimeSlimLayout)))
	op := NewFileOp()
	defer func() {
		_ = op.DeleteFile(tmpFile)
		if err != nil {
			_ = op.DeleteFile(dstFile)
		}
	}()
	baseDir := path.Dir(sourcePaths[0])
	relativePaths := make([]string, len(sourcePaths))
	for i, sp := range sourcePaths {
		relativePaths[i] = path.Base(sp)
	}
	// 使用 -r (非安静) 使 zip 逐行输出被添加的文件名
	cmdStr := fmt.Sprintf("zip -r %s  %s", tmpFile, strings.Join(relativePaths, " "))
	if err = cmd.ExecCmdWithStream(cmdStr, baseDir, outputFn); err != nil {
		return err
	}
	if err = op.Mv(tmpFile, dstFile); err != nil {
		return err
	}
	return nil
}