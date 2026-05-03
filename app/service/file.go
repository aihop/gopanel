package service

import (
	"errors"
	"fmt"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/dto/response"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/common"
	"github.com/aihop/gopanel/utils/files"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type FileService struct{ c fiber.Ctx }

func checkFilePermission(ctx fiber.Ctx, reqPaths ...string) error {
	if ctx == nil {
		return nil
	}
	claims, ok := ctx.Locals(constant.AppAuthName).(*token.CustomClaims)
	if !ok || claims == nil {
		return errors.New("unauthorized")
	}
	if claims.Role == constant.UserRoleSubAdmin {
		baseDir := filepath.Clean(claims.FileBaseDir)
		if baseDir == "" || baseDir == "/" || baseDir == "." {
			return errors.New("sub_admin account is not configured with a valid base directory")
		}
		for _, p := range reqPaths {
			cleanPath := filepath.Clean(p)
			if !strings.HasPrefix(cleanPath, baseDir) {
				global.LOG.Errorf("Permission Denied: SubAdmin %d tried to access %s (allowed: %s)", claims.UserId, cleanPath, baseDir)
				return errors.New("permission denied: you can only access your designated workspace")
			}
		}
	}
	return nil
}

type IFileService interface {
	GetFileList(op request.FileOption) (response.FileInfo, error)
	SearchUploadWithPage(req *request.SearchUploadWithPage) (int64, interface{}, error)
	GetFileTree(op request.FileOption) ([]response.FileTree, error)
	Create(op request.FileCreate) error
	Delete(op request.FileDelete) error
	BatchDelete(op request.FileBatchDelete) error
	Compress(c request.FileCompress) error
	DeCompress(c request.FileDeCompress) error
	GetContent(op request.FileContentReq) (response.FileInfo, error)
	SaveContent(edit request.FileEdit) error
	FileDownload(d request.FileDownload) (string, error)
	DirSize(req request.DirSizeReq) (response.DirSizeRes, error)
	ChangeName(req request.FileRename) error
	Wget(w request.FileWget) (string, error)
	MvFile(m request.FileMove) error
	ChangeOwner(req request.FileRoleUpdate) error
	ChangeMode(op request.FileCreate) error
	BatchChangeModeAndOwner(op request.FileRoleReq) error
	ReadLogByLine(req request.FileReadByLineReq) (*response.FileLineContent, error)
	BatchCheckFiles(req request.FilePathsCheck) []response.ExistFileInfo
}

var filteredPaths = []string{"/.gopanel_clash"}

func NewIFileService() *FileService {
	return &FileService{}
}
func (f *FileService) GetFileList(op request.FileOption) (response.FileInfo, error) {
	var fileInfo response.FileInfo
	data, err := os.Stat(op.Path)
	if err != nil && os.IsNotExist(err) {
		return fileInfo, nil
	}
	if !data.IsDir() {
		op.FileOption.Path = filepath.Dir(op.FileOption.Path)
	}
	info, err := files.NewFileInfo(op.FileOption)
	if err != nil {
		if isPermissionErr(err) {
			return f.getFileListViaGpc(op)
		}
		return response.FileInfo{}, err
	}
	fileInfo.FileInfo = *info
	return fileInfo, nil
}
func (f *FileService) SearchUploadWithPage(req *request.SearchUploadWithPage) (int64, interface{}, error) {
	var (
		files    []response.UploadInfo
		backData []response.UploadInfo
	)
	_ = filepath.Walk(req.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			files = append(files, response.UploadInfo{CreatedAt: info.ModTime().Format(constant.DateTimeLayout), Size: int(info.Size()), Name: info.Name()})
		}
		return nil
	})
	total, start, end := len(files), (req.Page-1)*req.Limit, req.Page*req.Limit
	if start > total {
		backData = make([]response.UploadInfo, 0)
	} else {
		if end >= total {
			end = total
		}
		backData = files[start:end]
	}
	return int64(total), backData, nil
}
func (f *FileService) Create(op request.FileCreate) error {
	if files.IsInvalidChar(op.Path) {
		return buserr.New("ErrInvalidChar")
	}
	fo := files.NewFileOp()
	if fo.Stat(op.Path) {
		return buserr.New(constant.ErrFileIsExist)
	}
	mode := op.Mode
	if mode == 0 {
		fileInfo, err := os.Stat(filepath.Dir(op.Path))
		if err == nil {
			mode = int64(fileInfo.Mode().Perm())
		} else {
			mode = 0755
		}
	}
	if op.IsDir {
		if err := fo.CreateDirWithMode(op.Path, fs.FileMode(mode)); err != nil {
			if isPermissionErr(err) {
				return f.mkdirViaGpc(op.Path, int(mode))
			}
			return err
		}
		return nil
	}
	if op.IsLink {
		if !fo.Stat(op.LinkPath) {
			return buserr.New(constant.ErrLinkPathNotFound)
		}
		return fo.LinkFile(op.LinkPath, op.Path, op.IsSymlink)
	}
	if err := fo.CreateFileWithMode(op.Path, fs.FileMode(mode)); err != nil {
		if isPermissionErr(err) {
			return f.createFileViaGpc(op.Path, "", int(mode))
		}
		return err
	}
	return nil
}
func (f *FileService) Delete(op request.FileDelete) error {
	if op.IsDir {
		excludeDir := global.CONF.System.BaseDir
		if filepath.Base(op.Path) == ".gopanel_clash" || op.Path == excludeDir {
			return buserr.New(constant.ErrPathNotDelete)
		}
	}
	fo := files.NewFileOp()
	if op.ForceDelete {
		if op.IsDir {
			if err := fo.DeleteDir(op.Path); err != nil {
				if isPermissionErr(err) {
					return f.removeViaGpc(op.Path)
				}
				return err
			}
			return nil
		} else {
			if err := fo.DeleteFile(op.Path); err != nil {
				if isPermissionErr(err) {
					return f.removeViaGpc(op.Path)
				}
				return err
			}
			return nil
		}
	}
	return nil
}
func (f *FileService) BatchDelete(op request.FileBatchDelete) error {
	fo := files.NewFileOp()
	if op.IsDir {
		for _, file := range op.Paths {
			if err := fo.DeleteDir(file); err != nil {
				if isPermissionErr(err) {
					if err2 := f.removeViaGpc(file); err2 != nil {
						return err2
					}
					continue
				}
				return err
			}
		}
	} else {
		for _, file := range op.Paths {
			if err := fo.DeleteFile(file); err != nil {
				if isPermissionErr(err) {
					if err2 := f.removeViaGpc(file); err2 != nil {
						return err2
					}
					continue
				}
				return err
			}
		}
	}
	return nil
}
func (f *FileService) ChangeMode(op request.FileCreate) error {
	fo := files.NewFileOp()
	if err := fo.ChmodR(op.Path, op.Mode, op.Sub); err != nil {
		if isPermissionErr(err) && !op.Sub {
			return f.chmodViaGpc(op.Path, int(op.Mode))
		}
		return err
	}
	return nil
}
func (f *FileService) BatchChangeModeAndOwner(op request.FileRoleReq) error {
	fo := files.NewFileOp()
	for _, path := range op.Paths {
		if !fo.Stat(path) {
			return buserr.New(constant.ErrPathNotFound)
		}
		if err := fo.ChownR(path, op.User, op.Group, op.Sub); err != nil {
			return err
		}
		if err := fo.ChmodR(path, op.Mode, op.Sub); err != nil {
			return err
		}
	}
	return nil
}
func (f *FileService) ChangeOwner(req request.FileRoleUpdate) error {
	fo := files.NewFileOp()
	if err := fo.ChownR(req.Path, req.User, req.Group, req.Sub); err != nil {
		if isPermissionErr(err) && !req.Sub {
			return f.chownViaGpc(req.Path, req.User, req.Group)
		}
		return err
	}
	return nil
}
func (f *FileService) Compress(c request.FileCompress) error {
	fo := files.NewFileOp()
	if !c.Replace && fo.Stat(filepath.Join(c.Dst, c.Name)) {
		return buserr.New(constant.ErrFileIsExist)
	}
	return fo.Compress(c.Files, c.Dst, c.Name, files.CompressType(c.Type), c.Secret)
}
func (f *FileService) DeCompress(c request.FileDeCompress) error {
	fo := files.NewFileOp()
	if c.Type == "tar" && len(c.Secret) != 0 {
		c.Type = "tar.gz"
	}
	return fo.Decompress(c.Path, c.Dst, files.CompressType(c.Type), c.Secret)
}
func (f *FileService) ChangeName(req request.FileRename) error {
	if files.IsInvalidChar(req.NewName) {
		return buserr.New("ErrInvalidChar")
	}
	fo := files.NewFileOp()
	return fo.Rename(req.OldName, req.NewName)
}
func (f *FileService) Wget(w request.FileWget) (string, error) {
	fo := files.NewFileOp()
	key := "file-wget-" + common.GetUuid()
	return key, fo.DownloadFileWithProcess(w.Url, filepath.Join(w.Path, w.Name), key, w.IgnoreCertificate)
}
func (f *FileService) MvFile(m request.FileMove) error {
	fo := files.NewFileOp()
	if !fo.Stat(m.NewPath) {
		return buserr.New(constant.ErrPathNotFound)
	}
	for _, oldPath := range m.OldPaths {
		if !fo.Stat(oldPath) {
			return buserr.WithNameNoCtx(constant.ErrFileNotFound, oldPath)
		}
		if oldPath == m.NewPath || strings.Contains(m.NewPath, filepath.Clean(oldPath)+"/") {
			return buserr.New(constant.ErrMovePathFailed)
		}
	}
	if m.Type == "cut" {
		return fo.Cut(m.OldPaths, m.NewPath, m.Name, m.Cover)
	}
	var errs []error
	if m.Type == "copy" {
		for _, src := range m.OldPaths {
			if err := fo.CopyAndReName(src, m.NewPath, m.Name, m.Cover); err != nil {
				errs = append(errs, err)
				global.LOG.Errorf("copy file [%s] to [%s] failed, err: %s", src, m.NewPath, err.Error())
			}
		}
	}
	var errString string
	for _, err := range errs {
		errString += err.Error() + "\n"
	}
	if errString != "" {
		return errors.New(errString)
	}
	return nil
}
func (f *FileService) FileDownload(d request.FileDownload) (string, error) {
	filePath := d.Paths[0]
	if d.Compress {
		tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().UnixNano()))
		if err := os.MkdirAll(tempPath, os.ModePerm); err != nil {
			return "", err
		}
		fo := files.NewFileOp()
		if err := fo.Compress(d.Paths, tempPath, d.Name, files.CompressType(d.Type), ""); err != nil {
			return "", err
		}
		filePath = filepath.Join(tempPath, d.Name)
	}
	return filePath, nil
}
func (f *FileService) DirSize(req request.DirSizeReq) (response.DirSizeRes, error) {
	var (
		res response.DirSizeRes
	)
	if req.Path == "/proc" {
		return res, nil
	}
	cmd := exec.Command("du", "-s", req.Path)
	output, err := cmd.Output()
	if err == nil {
		fields := strings.Fields(string(output))
		if len(fields) == 2 {
			var cmdSize int64
			_, err = fmt.Sscanf(fields[0], "%d", &cmdSize)
			if err == nil {
				res.Size = float64(cmdSize * 1024)
				return res, nil
			}
		}
	}
	fo := files.NewFileOp()
	size, err := fo.GetDirSize(req.Path)
	if err != nil {
		return res, err
	}
	res.Size = size
	return res, nil
}
func (f *FileService) ReadLogByLine(req request.FileReadByLineReq) (*response.FileLineContent, error) {
	logFilePath := ""
	switch req.Type {
	case "install":
		install_log_path := global.CONF.System.TmpDir + "/install_logs"
		logFilePath = path.Join(install_log_path, req.Name)
	case "image-pull", "image-push", "image-build", "compose-create":
		logFilePath = path.Join(global.CONF.System.TmpDir, fmt.Sprintf("docker_logs/%s", req.Name))
	case "ollama-model":
		logFilePath = path.Join(global.CONF.System.BaseDir, "log", "AITools", req.Name)
	case "mysql-slow-logs":
		logFilePath = path.Join(global.CONF.System.BaseDir, fmt.Sprintf("apps/mysql/%s/data/GoPanel-slow.log", req.Name))
	case "mariadb-slow-logs":
		logFilePath = path.Join(global.CONF.System.BaseDir, fmt.Sprintf("apps/mariadb/%s/db/data/GoPanel-slow.log", req.Name))
	}
	lines, isEndOfFile, total, err := files.ReadFileByLine(logFilePath, req.Page, req.Limit, req.Latest)
	if err != nil {
		return nil, err
	}
	if req.Latest && req.Page == 1 && len(lines) < 1000 && total > 1 {
		preLines, _, _, err := files.ReadFileByLine(logFilePath, total-1, req.Limit, false)
		if err != nil {
			return nil, err
		}
		lines = append(preLines, lines...)
	}
	res := &response.FileLineContent{Content: strings.Join(lines, "\n"), End: isEndOfFile, Path: logFilePath, Total: total, Lines: lines}
	return res, nil
}
func (f *FileService) BatchCheckFiles(req request.FilePathsCheck) []response.ExistFileInfo {
	fileList := make([]response.ExistFileInfo, 0, len(req.Paths))
	for _, filePath := range req.Paths {
		if info, err := os.Stat(filePath); err == nil {
			fileList = append(fileList, response.ExistFileInfo{Size: float64(info.Size()), Name: info.Name(), Path: filePath, ModTime: info.ModTime()})
		}
	}
	return fileList
}
