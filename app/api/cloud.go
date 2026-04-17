package api

import (
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/duke-git/lancet/v2/convertor"
	"github.com/gofiber/fiber/v3"
)

func CloudCdnDomains(c fiber.Ctx) error {
	id := c.Params("id")
	accountId, _ := convertor.ToInt(id)
	domains, err := service.NewCloudAccount().CdnDomains(uint(accountId))
	if err != nil {
		return c.JSON(e.Fail(buserr.Err(err)))
	}
	return c.JSON(e.Succ(domains))
}
