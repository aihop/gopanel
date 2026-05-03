package api

import (
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"time"
)

func refreshPreviewStatuses(sessionRepo repo.IAIDevSessionRepo, previews []*model.AIPreview) []*model.AIPreview {
	for _, preview := range previews {
		if preview == nil {
			continue
		}
		refreshed := refreshSinglePreviewStatus(sessionRepo, preview)
		if refreshed != nil {
			preview = refreshed
		}
	}
	return previews
}
func refreshSinglePreviewStatus(sessionRepo repo.IAIDevSessionRepo, preview *model.AIPreview) *model.AIPreview {
	if preview == nil {
		return nil
	}
	now := time.Now()
	if preview.LastCheckedAt != nil && now.Sub(*preview.LastCheckedAt) < 15*time.Second {
		return preview
	}
	parsed, err := neturl.Parse(preview.URL)
	if err != nil {
		preview.Status = "invalid"
		preview.LastCheckedAt = &now
		if updateErr := sessionRepo.UpdatePreview(preview); updateErr != nil {
			global.LOG.Warnf("update invalid preview status failed: %v", updateErr)
		}
		return preview
	}
	host := strings.ToLower(parsed.Hostname())
	switch host {
	case "localhost", "127.0.0.1", "0.0.0.0":
		preview.Status = "local"
		preview.LastCheckedAt = &now
		if updateErr := sessionRepo.UpdatePreview(preview); updateErr != nil {
			global.LOG.Warnf("update local preview status failed: %v", updateErr)
		}
		return preview
	}
	status := probePreviewStatus(preview.URL)
	preview.Status = status
	preview.LastCheckedAt = &now
	if updateErr := sessionRepo.UpdatePreview(preview); updateErr != nil {
		global.LOG.Warnf("update preview status failed: %v", updateErr)
	}
	return preview
}
func probePreviewStatus(previewURL string) string {
	req, err := http.NewRequest(http.MethodHead, previewURL, nil)
	if err == nil {
		resp, headErr := aiPreviewProbeClient.Do(req)
		if headErr == nil {
			defer resp.Body.Close()
			return previewStatusFromHTTP(resp.StatusCode)
		}
	}
	req, err = http.NewRequest(http.MethodGet, previewURL, nil)
	if err != nil {
		return "invalid"
	}
	resp, getErr := aiPreviewProbeClient.Do(req)
	if getErr != nil {
		return "unreachable"
	}
	defer resp.Body.Close()
	return previewStatusFromHTTP(resp.StatusCode)
}
func previewStatusFromHTTP(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 400:
		return "ready"
	case statusCode >= 400 && statusCode < 500:
		return "ready"
	case statusCode >= 500:
		return "unreachable"
	default:
		return "checking"
	}
}
func GetAISessionPreviews(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, _ := strconv.Atoi(c.Params("id"))
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	previews, err := repo.NewAIDevSessionRepo().GetPreviewsBySessionID(session.ID, 50)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	previews = refreshPreviewStatuses(repo.NewAIDevSessionRepo(), previews)
	return c.JSON(e.Succ(fiber.Map{"items": previews, "total": len(previews)}))
}
