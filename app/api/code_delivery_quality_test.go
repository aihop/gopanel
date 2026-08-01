package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func createQualityDeliverySession(t *testing.T, sessionID uint, command string) *model.AIDevSession {
	t.Helper()
	database := withCodeGovernanceDB(t)
	session, _ := createDeliveryWorktree(t, sessionID)
	session.ProjectID, session.Status = sessionID, codeSessionStatusActive
	project := &model.AIProject{ID: session.ProjectID, Name: "quality-delivery", CreatorID: session.UserID, RequireQualityGate: true}
	if err := database.Create(project).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(session).Error; err != nil {
		t.Fatal(err)
	}
	packageJSON := `{"scripts":{"test":"` + command + `"}}`
	if err := os.WriteFile(filepath.Join(session.WorkDir, "package.json"), []byte(packageJSON), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := saveCodeSessionWorktree(session, "test: configure quality gate"); err != nil {
		t.Fatal(err)
	}
	if _, err := persistCodeDeliveryJob(session, session.UserID, "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	return session
}

func TestRunCodeDeliveryQualityGateRunsMissingChecks(t *testing.T) {
	session := createQualityDeliverySession(t, 147, "true")
	if err := runCodeDeliveryQualityGate(session, session.UserID, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := validateCodeQualityGate(session); err != nil {
		t.Fatalf("automatic quality result did not satisfy gate: %v", err)
	}
}

func TestRunCodeDeliveryQualityGateStopsFailedDelivery(t *testing.T) {
	session := createQualityDeliverySession(t, 148, "false")
	err := runCodeDeliveryQualityGate(session, session.UserID, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "质量门禁未通过") || !strings.Contains(err.Error(), "> false") {
		t.Fatalf("failed quality check should stop delivery: %v", err)
	}
	checks, detectErr := detectCodeQualityChecks(session)
	if detectErr != nil || len(checks) != 1 {
		t.Fatalf("detect checks: %#v, %v", checks, detectErr)
	}
	loadCodeQualityResults(session.ID, checks)
	if checks[0].LastResult == nil || checks[0].LastResult.Status != "failed" {
		t.Fatalf("failed result was not persisted: %#v", checks[0].LastResult)
	}
}
