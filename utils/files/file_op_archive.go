package files

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	cZip "github.com/klauspost/compress/zip"
	"github.com/mholt/archiver/v4"
	"github.com/spf13/afero"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func getFormat(cType CompressType) archiver.CompressedArchive {
	format := archiver.CompressedArchive{}
	switch cType {
	case Tar:
		format.Archival = archiver.Tar{}
	case TarGz, Gz:
		format.Compression = archiver.Gz{}
		format.Archival = archiver.Tar{}
	case SdkTarGz:
		format.Compression = archiver.Gz{}
		format.Archival = archiver.Tar{}
	case SdkZip, Zip:
		format.Archival = archiver.Zip{
			Compression: zip.Deflate,
		}
	case Bz2:
		format.Compression = archiver.Bz2{}
		format.Archival = archiver.Tar{}
	case Xz:
		format.Compression = archiver.Xz{}
		format.Archival = archiver.Tar{}
	}
	return format
}

func (f FileOp) Compress(srcRiles []string, dst string, name string, cType CompressType, secret string) error {
	format := getFormat(cType)

	fileMaps := make(map[string]string, len(srcRiles))
	for _, s := range srcRiles {
		base := filepath.Base(s)
		fileMaps[s] = base
	}

	if !f.Stat(dst) {
		_ = f.CreateDir(dst, 0755)
	}

	files, err := archiver.FilesFromDisk(nil, fileMaps)
	if err != nil {
		return err
	}
	dstFile := filepath.Join(dst, name)
	out, err := f.Fs.Create(dstFile)
	if err != nil {
		return err
	}

	switch cType {
	case Zip:
		if err := ZipFile(files, out); err == nil {
			return nil
		}
		_ = f.DeleteFile(dstFile)
		return NewZipArchiver().Compress(srcRiles, dstFile, "")
	case TarGz:
		err = NewTarGzArchiver().Compress(srcRiles, dstFile, secret)
		if err != nil {
			_ = f.DeleteFile(dstFile)
			return err
		}
	default:
		err = format.Archive(context.Background(), out, files)
		if err != nil {
			_ = f.DeleteFile(dstFile)
			return err
		}
	}
	return nil
}

// CompressWithCallback 执行压缩并通过 progressFn 回传进度消息。
func (f FileOp) CompressWithCallback(srcFiles []string, dst string, name string, cType CompressType, secret string, progressFn func(msg string)) error {
	if progressFn != nil {
		progressFn(fmt.Sprintf("开始打包 %d 个文件...", len(srcFiles)))
	}

	fileMaps := make(map[string]string, len(srcFiles))
	for _, s := range srcFiles {
		base := filepath.Base(s)
		fileMaps[s] = base
	}

	if !f.Stat(dst) {
		_ = f.CreateDir(dst, 0755)
	}

	dstFile := filepath.Join(dst, name)

	switch cType {
	case Zip:
		// 先尝试 SDK 方式
		files, err := archiver.FilesFromDisk(nil, fileMaps)
		if err == nil {
			out, err := f.Fs.Create(dstFile)
			if err == nil {
				if err := ZipFile(files, out); err == nil {
					if progressFn != nil {
						progressFn(fmt.Sprintf("打包完成，共 %d 个文件", len(srcFiles)))
					}
					return nil
				}
				_ = out.Close()
				_ = f.DeleteFile(dstFile)
			}
		}
		// SDK 失败，回退到 shell zip（带 verbose 输出）
		if progressFn != nil {
			progressFn("SDK 打包失败，尝试系统 zip 命令...")
		}
		outputFn := func(line string) {
			if progressFn != nil {
				progressFn(line)
			}
		}
		if err := NewZipArchiver().(*ZipArchiver).CompressWithOutput(srcFiles, dstFile, "", outputFn); err != nil {
			_ = f.DeleteFile(dstFile)
			return err
		}
	case TarGz:
		outputFn := func(line string) {
			if progressFn != nil {
				progressFn(line)
			}
		}
		if err := NewTarGzArchiver().(*TarGzArchiver).CompressWithOutput(srcFiles, dstFile, secret, outputFn); err != nil {
			_ = f.DeleteFile(dstFile)
			return err
		}
	default:
		// Tar、Bz2、Xz 等使用 SDK 方式
		format := getFormat(cType)
		files, err := archiver.FilesFromDisk(nil, fileMaps)
		if err != nil {
			return err
		}
		out, err := f.Fs.Create(dstFile)
		if err != nil {
			return err
		}
		if err = format.Archive(context.Background(), out, files); err != nil {
			_ = f.DeleteFile(dstFile)
			return err
		}
	}

	if progressFn != nil {
		progressFn(fmt.Sprintf("打包完成：%s", dstFile))
	}
	return nil
}

func isIgnoreFile(name string) bool {
	return strings.HasPrefix(name, "__MACOSX") || strings.HasSuffix(name, ".DS_Store") || strings.HasPrefix(name, "._")
}

func decodeGBK(input string) (string, error) {
	decoder := simplifiedchinese.GBK.NewDecoder()
	decoded, _, err := transform.String(decoder, input)
	if err != nil {
		return "", err
	}
	return decoded, nil
}

func (f FileOp) decompressWithSDK(srcFile string, dst string, cType CompressType) error {
	format := getFormat(cType)
	handler := func(ctx context.Context, archFile archiver.File) error {
		info := archFile.FileInfo
		if isIgnoreFile(archFile.Name()) {
			return nil
		}
		fileName := archFile.NameInArchive
		var err error
		if header, ok := archFile.Header.(cZip.FileHeader); ok {
			if header.NonUTF8 && header.Flags == 0 {
				fileName, err = decodeGBK(fileName)
				if err != nil {
					return err
				}
			}
		}
		filePath := filepath.Join(dst, fileName)
		if archFile.FileInfo.IsDir() {
			if err := f.Fs.MkdirAll(filePath, info.Mode()); err != nil {
				return err
			}
			return nil
		} else {
			parentDir := path.Dir(filePath)
			if !f.Stat(parentDir) {
				if err := f.Fs.MkdirAll(parentDir, info.Mode()); err != nil {
					return err
				}
			}
		}
		fr, err := archFile.Open()
		if err != nil {
			return err
		}
		defer fr.Close()
		fw, err := f.Fs.OpenFile(filePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer fw.Close()
		if _, err := io.Copy(fw, fr); err != nil {
			return err
		}

		return nil
	}
	input, err := f.Fs.Open(srcFile)
	if err != nil {
		return err
	}
	return format.Extract(context.Background(), input, nil, handler)
}

func (f FileOp) Decompress(srcFile string, dst string, cType CompressType, secret string) error {
	if cType == Tar || cType == Zip || cType == TarGz {
		shellArchiver, err := NewShellArchiver(cType)
		if !f.Stat(dst) {
			_ = f.CreateDir(dst, 0755)
		}
		if err == nil {
			if err = shellArchiver.Extract(srcFile, dst, secret); err == nil {
				return nil
			}
		}
	}
	return f.decompressWithSDK(srcFile, dst, cType)
}

func ZipFile(files []archiver.File, dst afero.File) error {
	zw := zip.NewWriter(dst)
	defer zw.Close()

	for _, file := range files {
		hdr, err := zip.FileInfoHeader(file)
		if err != nil {
			return err
		}
		hdr.Method = zip.Deflate
		hdr.Name = file.NameInArchive
		if file.IsDir() {
			if !strings.HasSuffix(hdr.Name, "/") {
				hdr.Name += "/"
			}
		}
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		if file.IsDir() {
			continue
		}

		if file.LinkTarget != "" {
			_, err = w.Write([]byte(filepath.ToSlash(file.LinkTarget)))
			if err != nil {
				return err
			}
		} else {
			fileReader, err := file.Open()
			if err != nil {
				return err
			}
			_, err = io.Copy(w, fileReader)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
