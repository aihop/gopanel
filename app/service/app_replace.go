package service

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/aihop/gopanel/global"
)

func replace1PanelIdentifiers(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yml" || ext == ".yaml" || ext == ".env" || ext == ".sh" || ext == ".conf" || ext == ".json" || ext == ".txt" || ext == "" || ext == ".md" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			text := string(content)
			original := text

			// Replace sequences carefully
			text = strings.ReplaceAll(text, "1panel-network", "gopanel-network")
			text = strings.ReplaceAll(text, "/opt/1panel", "/opt/gopanel")
			text = strings.ReplaceAll(text, "1panel", "gopanel")
			text = strings.ReplaceAll(text, "1Panel", "GoPanel")

			if original != text {
				global.LOG.Debugf("Replaced 1Panel identifiers in file: %s", path)
				return os.WriteFile(path, []byte(text), info.Mode())
			}
		}
		return nil
	})
}
