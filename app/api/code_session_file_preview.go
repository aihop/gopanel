package api

import (
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/gofiber/fiber/v3"
)

const maxAISessionImagePreviewSize = 4 * 1024 * 1024

func conversationImageContentType(relativePath string) string {
	switch strings.ToLower(filepath.Ext(relativePath)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}

func readAISessionImagePreview(workDir, relativePath string, sourceDirs []string) (fiber.Map, error) {
	target, cleanRelative, info, err := resolveAISessionRegularFile(workDir, relativePath, sourceDirs)
	if err != nil {
		return nil, err
	}
	contentType := conversationImageContentType(cleanRelative)
	if contentType == "" {
		return nil, errors.New("仅支持预览 PNG、JPEG、GIF 或 WebP 图片")
	}
	if info.Size() > maxAISessionImagePreviewSize {
		return nil, errors.New("图片超过 4 MB，无法在对话中预览")
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	return fiber.Map{
		"path":        cleanRelative,
		"contentType": contentType,
		"content":     base64.StdEncoding.EncodeToString(content),
		"size":        len(content),
	}, nil
}

func GetAISessionFilePreview(c fiber.Ctx) error {
	_, workDir, sourceDirs, err := getAISessionFileContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	result, err := readAISessionImagePreview(workDir, c.Query("path"), sourceDirs)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(result))
}
