package api

import (
	"context"
	"strconv"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/utils/convertor"
	"github.com/gofiber/fiber/v3"
)

func websiteDiagnosticIDs(c fiber.Ctx) (uint, uint, error) {
	websiteID := convertor.ToUint(c.Params("id"))
	issueID := convertor.ToUint(c.Params("issueId"))
	if websiteID == 0 {
		return 0, 0, buserr.New("ErrWebsiteDiagnosticInvalidWebsiteID")
	}
	return websiteID, issueID, nil
}

func ListWebsiteDiagnosticIssues(c fiber.Ctx) error {
	websiteID, _, err := websiteDiagnosticIDs(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	items, total, err := service.ListWebsiteIssues(websiteID, c.Query("status", "all"), page, limit)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"items": items, "total": total}))
}

func GetWebsiteDiagnosticIssue(c fiber.Ctx) error {
	websiteID, issueID, err := websiteDiagnosticIDs(c)
	if err != nil || issueID == 0 {
		return c.JSON(e.Fail(buserr.New("ErrWebsiteDiagnosticIssueNotFound")))
	}
	detail, err := service.GetWebsiteIssueDetail(websiteID, issueID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(detail))
}

func UpdateWebsiteDiagnosticIssue(c fiber.Ctx) error {
	websiteID, issueID, err := websiteDiagnosticIDs(c)
	if err != nil || issueID == 0 {
		return c.JSON(e.Fail(buserr.New("ErrWebsiteDiagnosticIssueNotFound")))
	}
	var input struct {
		Action string `json:"action"`
	}
	if err = c.Bind().JSON(&input); err != nil {
		return c.JSON(e.Fail(err))
	}
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	issue, err := service.UpdateWebsiteIssueStatus(websiteID, issueID, claims.UserId, input.Action)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(issue))
}

func HandoffWebsiteDiagnosticIssue(c fiber.Ctx) error {
	websiteID, issueID, err := websiteDiagnosticIDs(c)
	if err != nil || issueID == 0 {
		return c.JSON(e.Fail(buserr.New("ErrWebsiteDiagnosticIssueNotFound")))
	}
	var input websiteIssueCodeRequest
	if err = c.Bind().JSON(&input); err != nil {
		return c.JSON(e.Fail(err))
	}
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	issue, err := handoffWebsiteIssueToCode(websiteID, issueID, claims, input)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(issue))
}

func VerifyWebsiteDiagnosticIssue(c fiber.Ctx) error {
	websiteID, issueID, err := websiteDiagnosticIDs(c)
	if err != nil || issueID == 0 {
		return c.JSON(e.Fail(buserr.New("ErrWebsiteDiagnosticIssueNotFound")))
	}
	var input struct {
		Release string `json:"release"`
	}
	if err = c.Bind().JSON(&input); err != nil {
		return c.JSON(e.Fail(err))
	}
	claims, err := middleware.JwtClaims(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	issue, err := service.MarkWebsiteIssueVerifying(websiteID, issueID, claims.UserId, input.Release)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(issue))
}

func ListWebsiteDiagnosticProbes(c fiber.Ctx) error {
	websiteID, _, err := websiteDiagnosticIDs(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	probes, err := service.ListWebsiteProbes(websiteID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(probes))
}

func SaveWebsiteDiagnosticProbes(c fiber.Ctx) error {
	websiteID, _, err := websiteDiagnosticIDs(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var probes []model.WebsiteProbe
	if err = c.Bind().JSON(&probes); err != nil {
		return c.JSON(e.Fail(err))
	}
	probes, err = service.SaveWebsiteProbes(websiteID, probes)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(probes))
}

func RunWebsiteDiagnosticProbe(c fiber.Ctx) error {
	websiteID, _, err := websiteDiagnosticIDs(c)
	probeID := convertor.ToUint(c.Params("probeId"))
	if err != nil || probeID == 0 {
		return c.JSON(e.Fail(buserr.New("ErrWebsiteDiagnosticProbeNotFound")))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	probe, err := service.RunWebsiteProbeNow(ctx, websiteID, probeID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(probe))
}
