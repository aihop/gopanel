package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/global"
)

const websiteDiagnosticProbeBodyLimit = 256 * 1024

func normalizeWebsiteProbes(probes []model.WebsiteProbe) error {
	if len(probes) > 20 {
		return buserr.New("ErrWebsiteDiagnosticTooManyProbes")
	}
	for index := range probes {
		probe := &probes[index]
		probe.Name = limitedDiagnosticText(probe.Name, 128)
		probe.Method = strings.ToUpper(limitedDiagnosticText(probe.Method, 16))
		probe.Path = limitedDiagnosticText(probe.Path, 512)
		if probe.Name == "" || !strings.HasPrefix(probe.Path, "/") {
			return buserr.New("ErrWebsiteDiagnosticInvalidProbe")
		}
		if probe.Method != http.MethodGet && probe.Method != http.MethodHead {
			return buserr.New("ErrWebsiteDiagnosticInvalidProbeMethod")
		}
		if probe.ExpectedStatus < 100 || probe.ExpectedStatus > 599 {
			return buserr.New("ErrWebsiteDiagnosticInvalidProbeStatus")
		}
		if probe.TimeoutMS < 100 || probe.TimeoutMS > 30000 || probe.IntervalSeconds < 10 || probe.IntervalSeconds > 86400 || probe.FailureThreshold < 1 || probe.FailureThreshold > 100 {
			return buserr.New("ErrWebsiteDiagnosticInvalidProbeRange")
		}
		fields := splitProbeFields(probe.RequiredFields)
		probe.RequiredFields = strings.Join(fields, ",")
		probe.ExpectedCode = limitedDiagnosticText(probe.ExpectedCode, 128)
	}
	return nil
}

func saveWebsiteProbes(websiteID uint, probes []model.WebsiteProbe) ([]model.WebsiteProbe, error) {
	if _, err := repo.NewWebsite().GetFirst(repo.NewCommonRepo().WithByID(websiteID)); err != nil {
		return nil, buserr.New("ErrWebsiteDiagnosticWebsiteNotFound")
	}
	if err := normalizeWebsiteProbes(probes); err != nil {
		return nil, err
	}
	repository := repo.NewWebsiteDiagnostic(global.DB)
	if err := repository.SaveProbes(websiteID, probes); err != nil {
		return nil, err
	}
	return repository.ListProbes(websiteID)
}

func runDueWebsiteProbes(ctx context.Context) error {
	probes, err := repo.NewWebsiteDiagnostic(global.DB).ListDueProbes(time.Now())
	if err != nil {
		return err
	}
	var runErrors []error
	for index := range probes {
		if err = runWebsiteProbe(ctx, &probes[index]); err != nil {
			runErrors = append(runErrors, err)
		}
	}
	return errors.Join(runErrors...)
}

func runWebsiteProbesForWebsite(ctx context.Context, websiteID uint) error {
	probes, err := repo.NewWebsiteDiagnostic(global.DB).ListProbes(websiteID)
	if err != nil {
		return err
	}
	var runErrors []error
	for index := range probes {
		if probes[index].Enabled {
			if runErr := runWebsiteProbe(ctx, &probes[index]); runErr != nil {
				runErrors = append(runErrors, runErr)
			}
		}
	}
	return errors.Join(runErrors...)
}

func runWebsiteProbe(ctx context.Context, probe *model.WebsiteProbe) error {
	website, err := repo.NewWebsite().GetFirst(repo.NewCommonRepo().WithByID(probe.WebsiteID))
	if err != nil {
		return err
	}
	setting, err := repo.NewWebsiteDiagnostic(global.DB).GetByWebsiteID(probe.WebsiteID)
	if err != nil || setting == nil || !setting.Enabled || !setting.ActiveProbes {
		return err
	}
	protocol := strings.ToLower(normalizeWebsiteProtocol(website.Protocol))
	if protocol == "" {
		protocol = "https"
	}
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(probe.TimeoutMS)*time.Millisecond)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, probe.Method, protocol+"://"+website.PrimaryDomain+probe.Path, nil)
	if err != nil {
		return err
	}
	started := time.Now()
	response, requestErr := http.DefaultClient.Do(request)
	duration := time.Since(started)
	message := ""
	passed := requestErr == nil
	var payload interface{}
	status := 0
	if requestErr != nil {
		message = requestErr.Error()
	} else {
		defer response.Body.Close()
		status = response.StatusCode
		body, readErr := io.ReadAll(io.LimitReader(response.Body, websiteDiagnosticProbeBodyLimit+1))
		if readErr != nil {
			passed, message = false, readErr.Error()
		}
		if len(body) > websiteDiagnosticProbeBodyLimit {
			passed, message = false, "response body too large"
		}
		if passed && probe.ExpectedStatus != status {
			passed, message = false, fmt.Sprintf("expected HTTP %d, received %d", probe.ExpectedStatus, status)
		}
		if len(body) > 0 {
			_ = json.Unmarshal(body, &payload)
		}
		if passed && probe.ExpectedCode != "" && fmt.Sprint(jsonPathValue(payload, "code")) != probe.ExpectedCode {
			passed, message = false, "business code mismatch"
		}
		if passed {
			for _, field := range splitProbeFields(probe.RequiredFields) {
				if jsonPathValue(payload, field) == nil {
					passed, message = false, "missing field: "+field
					break
				}
			}
		}
	}
	now := time.Now()
	updates := map[string]interface{}{"last_run_at": now, "updated_at": now}
	if passed {
		updates["last_status"], updates["last_message"], updates["failure_count"] = "success", "", 0
	} else {
		probe.FailureCount++
		updates["last_status"], updates["last_message"], updates["failure_count"] = "failed", limitedDiagnosticText(message, 2048), probe.FailureCount
	}
	if err = global.DB.Model(probe).Updates(updates).Error; err != nil {
		return err
	}
	if !passed && probe.FailureCount >= probe.FailureThreshold {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%d", probe.ID, now.Unix(), probe.FailureCount)))
		_, _, err = ingestWebsiteDiagnosticEnvelope(website.ID, &WebsiteDiagnosticEnvelope{
			Schema: websiteDiagnosticSchema, EventID: "probe-" + hex.EncodeToString(sum[:12]), WebsiteID: website.ID,
			Source: "probe", Kind: "probe_failed", Severity: "error", Title: probe.Name,
			Message: message, Method: probe.Method, Route: probe.Path, HTTPStatus: status,
			DurationMS: duration.Milliseconds(), Release: activeWebsiteRelease(website.ID), OccurredAt: now,
		})
	}
	return err
}

func RunWebsiteProbeNow(ctx context.Context, websiteID, probeID uint) (*model.WebsiteProbe, error) {
	var probe model.WebsiteProbe
	if err := global.DB.Where("id = ? AND website_id = ?", probeID, websiteID).First(&probe).Error; err != nil {
		return nil, buserr.New("ErrWebsiteDiagnosticProbeNotFound")
	}
	if err := runWebsiteProbe(ctx, &probe); err != nil {
		return nil, err
	}
	if err := global.DB.First(&probe, probe.ID).Error; err != nil {
		return nil, err
	}
	return &probe, nil
}

func ListWebsiteProbes(websiteID uint) ([]model.WebsiteProbe, error) {
	if _, err := repo.NewWebsite().GetFirst(repo.NewCommonRepo().WithByID(websiteID)); err != nil {
		return nil, buserr.New("ErrWebsiteDiagnosticWebsiteNotFound")
	}
	return repo.NewWebsiteDiagnostic(global.DB).ListProbes(websiteID)
}

func SaveWebsiteProbes(websiteID uint, probes []model.WebsiteProbe) ([]model.WebsiteProbe, error) {
	return saveWebsiteProbes(websiteID, probes)
}

func splitProbeFields(value string) []string {
	parts := strings.FieldsFunc(value, func(char rune) bool { return char == ',' || char == '\n' })
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, limitedDiagnosticText(part, 128))
		}
	}
	return result
}

func jsonPathValue(value interface{}, path string) interface{} {
	current := value
	for _, segment := range strings.Split(path, ".") {
		object, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current, ok = object[segment]
		if !ok {
			return nil
		}
	}
	return current
}

func activeWebsiteRelease(websiteID uint) string {
	if !global.DB.Migrator().HasTable(&model.AppDeploy{}) {
		return ""
	}
	var deploy model.AppDeploy
	if err := global.DB.Where("website_id = ? AND is_active = ?", websiteID, true).Order("updated_at DESC").First(&deploy).Error; err != nil {
		return ""
	}
	return limitedDiagnosticText(deploy.Version, 128)
}
