package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/global"
)

type websiteDiagnosticRuntime struct {
	once   sync.Once
	cancel context.CancelFunc
	done   chan struct{}
}

var diagnosticRuntime = &websiteDiagnosticRuntime{done: make(chan struct{})}

var websiteDiagnosticLastCleanup time.Time

func StartWebsiteDiagnosticRuntime() {
	diagnosticRuntime.once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		diagnosticRuntime.cancel = cancel
		go func() {
			defer close(diagnosticRuntime.done)
			runWebsiteDiagnostics(ctx)
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					runWebsiteDiagnostics(ctx)
				case <-ctx.Done():
					return
				}
			}
		}()
	})
}

func ShutdownWebsiteDiagnosticRuntime(ctx context.Context) error {
	if diagnosticRuntime.cancel == nil {
		return nil
	}
	diagnosticRuntime.cancel()
	select {
	case <-diagnosticRuntime.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func runWebsiteDiagnostics(ctx context.Context) {
	if global.DB == nil {
		return
	}
	settings, err := repo.NewWebsiteDiagnostic(global.DB).ListEnabled()
	if err != nil {
		logWebsiteDiagnosticRuntimeError("load settings", err)
		return
	}
	var runErrors []error
	for index := range settings {
		setting := &settings[index]
		website, findErr := repo.NewWebsite().GetFirst(repo.NewCommonRepo().WithByID(setting.WebsiteID))
		if findErr != nil {
			runErrors = append(runErrors, findErr)
			continue
		}
		if collectErr := collectWebsiteCaddyEvents(&website, setting); collectErr != nil {
			runErrors = append(runErrors, collectErr)
		}
	}
	if consumeErr := NewWebsiteDiagnosticConsumer().RunOnce(); consumeErr != nil {
		runErrors = append(runErrors, consumeErr)
	}
	if probeErr := runDueWebsiteProbes(ctx); probeErr != nil {
		runErrors = append(runErrors, probeErr)
	}
	if verifyErr := ReconcileWebsiteIssueVerification(time.Now()); verifyErr != nil {
		runErrors = append(runErrors, verifyErr)
	}
	if deployErr := ReconcileWebsiteIssueDeployments(); deployErr != nil {
		runErrors = append(runErrors, deployErr)
	}
	if time.Since(websiteDiagnosticLastCleanup) >= time.Hour {
		configured, loadErr := repo.NewWebsiteDiagnostic(global.DB).ListConfigured()
		if loadErr != nil {
			runErrors = append(runErrors, loadErr)
		} else if cleanupErr := cleanupWebsiteDiagnosticRetention(configured, time.Now()); cleanupErr != nil {
			runErrors = append(runErrors, cleanupErr)
		} else {
			websiteDiagnosticLastCleanup = time.Now()
		}
	}
	if err = errors.Join(runErrors...); err != nil {
		logWebsiteDiagnosticRuntimeError("run", err)
	}
}

func cleanupWebsiteDiagnosticRetention(settings []model.WebsiteDiagnosticSetting, now time.Time) error {
	var cleanupErrors []error
	for _, setting := range settings {
		cutoff := now.AddDate(0, 0, -setting.RetentionDays)
		if err := global.DB.Where("website_id = ? AND occurred_at < ?", setting.WebsiteID, cutoff).Delete(&model.WebsiteDiagnosticEvent{}).Error; err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
		website, err := repo.NewWebsite().GetFirst(repo.NewCommonRepo().WithByID(setting.WebsiteID))
		if err != nil {
			continue
		}
		trackingDir, err := websiteTrackingDir(website.Alias)
		if err != nil {
			continue
		}
		for _, folder := range []string{"processed", "rejected"} {
			entries, readErr := os.ReadDir(filepath.Join(trackingDir, folder))
			if readErr != nil {
				continue
			}
			for _, entry := range entries {
				info, statErr := entry.Info()
				if statErr == nil && info.ModTime().Before(cutoff) {
					_ = os.Remove(filepath.Join(trackingDir, folder, entry.Name()))
				}
			}
		}
	}
	return errors.Join(cleanupErrors...)
}

func logWebsiteDiagnosticRuntimeError(operation string, err error) {
	if err != nil && global.LOG != nil {
		global.LOG.Errorf("Website diagnostics %s failed: %v", operation, err)
	}
}
