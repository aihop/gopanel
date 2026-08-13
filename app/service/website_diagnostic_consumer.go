package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
)

const websiteDiagnosticMaxEventBytes = 256 * 1024

type WebsiteDiagnosticConsumer struct {
	mu sync.Mutex
}

func NewWebsiteDiagnosticConsumer() *WebsiteDiagnosticConsumer { return &WebsiteDiagnosticConsumer{} }

func (consumer *WebsiteDiagnosticConsumer) RunOnce() error {
	consumer.mu.Lock()
	defer consumer.mu.Unlock()
	settings, err := repo.NewWebsiteDiagnostic(global.DB).ListEnabled()
	if err != nil {
		return err
	}
	var runErrors []error
	for _, setting := range settings {
		if !setting.BackendHook && !setting.BrowserHook {
			continue
		}
		website, findErr := repo.NewWebsite().GetFirst(repo.NewCommonRepo().WithByID(setting.WebsiteID))
		if findErr != nil {
			runErrors = append(runErrors, findErr)
			continue
		}
		if processErr := consumer.processWebsite(&website); processErr != nil {
			runErrors = append(runErrors, processErr)
		}
	}
	return errors.Join(runErrors...)
}

func (consumer *WebsiteDiagnosticConsumer) processWebsite(website *model.Website) error {
	trackingDir, err := ensureWebsiteTrackingDirs(website.Alias)
	if err != nil {
		return err
	}
	if err = consumer.recoverProcessing(trackingDir, website.ID); err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Join(trackingDir, "inbox"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ready") {
			continue
		}
		source := filepath.Join(trackingDir, "inbox", entry.Name())
		claimed := filepath.Join(trackingDir, "processing", entry.Name())
		if err = os.Rename(source, claimed); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		consumer.consumeClaimed(trackingDir, claimed, website.ID)
	}
	return nil
}

func (consumer *WebsiteDiagnosticConsumer) recoverProcessing(trackingDir string, websiteID uint) error {
	entries, err := os.ReadDir(filepath.Join(trackingDir, "processing"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ready") {
			continue
		}
		consumer.consumeClaimed(trackingDir, filepath.Join(trackingDir, "processing", entry.Name()), websiteID)
	}
	return nil
}

func (consumer *WebsiteDiagnosticConsumer) consumeClaimed(trackingDir, claimed string, websiteID uint) {
	destination := "processed"
	if err := consumeWebsiteDiagnosticFile(claimed, websiteID); err != nil {
		destination = "rejected"
		_ = os.WriteFile(claimed+".error", []byte(limitedDiagnosticText(err.Error(), 2048)), 0640)
	}
	target := filepath.Join(trackingDir, destination, filepath.Base(claimed))
	if err := os.Rename(claimed, target); err != nil && global.LOG != nil {
		global.LOG.Errorf("Move website diagnostic event failed: %v", err)
	}
	if destination == "rejected" {
		_ = os.Rename(claimed+".error", target+".error")
	}
}

func consumeWebsiteDiagnosticFile(path string, websiteID uint) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, websiteDiagnosticMaxEventBytes+1))
	if err != nil {
		return err
	}
	if len(data) > websiteDiagnosticMaxEventBytes {
		return fmt.Errorf("event exceeds %d bytes", websiteDiagnosticMaxEventBytes)
	}
	var envelope WebsiteDiagnosticEnvelope
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	if err = decoder.Decode(&envelope); err != nil {
		return err
	}
	_, _, err = ingestWebsiteDiagnosticEnvelope(websiteID, &envelope)
	return err
}

func WriteWebsiteDiagnosticEvent(alias string, event WebsiteDiagnosticEnvelope) (string, error) {
	trackingDir, err := ensureWebsiteTrackingDirs(alias)
	if err != nil {
		return "", err
	}
	if event.Schema == "" {
		event.Schema = websiteDiagnosticSchema
	}
	data, err := json.Marshal(event)
	if err != nil {
		return "", err
	}
	if len(data) > websiteDiagnosticMaxEventBytes {
		return "", fmt.Errorf("event exceeds %d bytes", websiteDiagnosticMaxEventBytes)
	}
	name := limitedDiagnosticText(event.EventID, 96)
	name = strings.Map(func(char rune) rune {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_' {
			return char
		}
		return '-'
	}, name)
	if name == "" {
		return "", errors.New("event ID is required")
	}
	tmp := filepath.Join(trackingDir, "inbox", name+".tmp")
	ready := filepath.Join(trackingDir, "inbox", name+".ready")
	if err = os.WriteFile(tmp, data, 0640); err != nil {
		return "", err
	}
	if err = os.Rename(tmp, ready); err != nil {
		return "", err
	}
	return ready, nil
}
