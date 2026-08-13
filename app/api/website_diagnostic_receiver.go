package api

import (
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/utils/convertor"
	"github.com/gofiber/fiber/v3"
)

func RotateWebsiteDiagnosticHookSecret(c fiber.Ctx) error {
	websiteID := convertor.ToUint(c.Params("id"))
	if websiteID == 0 {
		return c.JSON(e.Fail(buserr.New("ErrWebsiteDiagnosticInvalidWebsiteID")))
	}
	secret, err := service.RotateWebsiteDiagnosticHookSecret(websiteID)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"secret": secret}))
}

func ReceiveWebsiteDiagnosticEvent(c fiber.Ctx) error {
	websiteID := convertor.ToUint(c.Params("id"))
	if websiteID == 0 {
		return c.JSON(e.Fail(buserr.New("ErrWebsiteDiagnosticInvalidWebsiteID")))
	}
	issue, err := service.ReceiveWebsiteDiagnosticEvent(
		websiteID, c.Path(), c.Get("X-GoPanel-Timestamp"), c.Get("X-GoPanel-Nonce"),
		c.Get("X-GoPanel-Signature"), c.Get("Origin"), c.IP(), c.Body(),
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"accepted": true, "issueId": issue.ID}))
}
