package api

import (
	"context"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/dto/request"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/pkg/gormx"
	"github.com/gofiber/fiber/v3"
)

type mobileWebsiteSummary struct {
	ID                       uint   `json:"id"`
	Alias                    string `json:"alias"`
	PrimaryDomain            string `json:"primaryDomain"`
	OtherDomains             string `json:"otherDomains"`
	Protocol                 string `json:"protocol"`
	Type                     string `json:"type"`
	Status                   string `json:"status"`
	AppName                  string `json:"appName"`
	RedirectDomainsToPrimary bool   `json:"redirectDomainsToPrimary"`
}

type mobileDatabaseSummary struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Server   string `json:"server"`
	Encoding string `json:"encoding"`
	Comment  string `json:"comment"`
}

type mobileSSLSummary struct {
	ID            uint      `json:"id"`
	PrimaryDomain string    `json:"primaryDomain"`
	Type          string    `json:"type"`
	Provider      string    `json:"provider"`
	Status        string    `json:"status"`
	AutoRenew     bool      `json:"autoRenew"`
	ExpireDate    time.Time `json:"expireDate"`
}

type mobileAppSummary struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	Description string `json:"description"`
	HTTPPort    int    `json:"httpPort"`
	HTTPSPort   int    `json:"httpsPort"`
	RuntimeHost string `json:"runtimeHost"`
	RuntimeKind string `json:"runtimeKind"`
}

func GetMobileWebsites(c fiber.Ctx) error {
	websites, err := service.NewWebsite().List(gormx.NewContextx(200, "id desc"))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	items := make([]mobileWebsiteSummary, 0, len(websites))
	for _, website := range websites {
		items = append(items, mobileWebsiteSummary{
			ID: website.ID, Alias: website.Alias, PrimaryDomain: website.PrimaryDomain,
			OtherDomains: website.OtherDomains, Protocol: website.Protocol, Type: website.Type,
			Status: website.Status, AppName: website.AppName,
			RedirectDomainsToPrimary: website.RedirectDomainsToPrimary,
		})
	}
	return c.JSON(e.Succ(fiber.Map{"items": items, "total": len(items)}))
}

func UpdateMobileWebsiteDomainBindings(c fiber.Ctx) error {
	req, err := e.BodyToStruct[request.WebsiteDomainBindingUpdate](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := service.NewWebsite().UpdateDomainBindings(
		ctx,
		req.WebsiteID,
		req.PrimaryDomain,
		req.OtherDomains,
		req.RedirectDomainsToPrimary,
	); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func GetMobileDatabases(c fiber.Ctx) error {
	ctx := gormx.NewContextx(200, "name asc")
	ctx.Page = 1
	result, err := service.NewDatabase().List(ctx)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	items := make([]mobileDatabaseSummary, 0, len(result.Items))
	for _, database := range result.Items {
		items = append(items, mobileDatabaseSummary{
			Type: string(database.Type), Name: database.Name, Server: database.Server,
			Encoding: database.Encoding, Comment: database.Comment,
		})
	}
	return c.JSON(e.Succ(fiber.Map{"items": items, "total": result.Total, "warningCount": len(result.Warnings)}))
}

func GetMobileSSLs(c fiber.Ctx) error {
	certificates, err := service.NewSSL().List(gormx.NewContextx(200, "expire_date asc"))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	items := make([]mobileSSLSummary, 0, len(certificates))
	for _, certificate := range certificates {
		items = append(items, mobileSSLSummary{
			ID: certificate.ID, PrimaryDomain: certificate.PrimaryDomain, Type: certificate.Type,
			Provider: certificate.Provider, Status: certificate.Status, AutoRenew: certificate.AutoRenew,
			ExpireDate: certificate.ExpireDate,
		})
	}
	return c.JSON(e.Succ(fiber.Map{"items": items, "total": len(items)}))
}

func GetMobileApps(c fiber.Ctx) error {
	total, apps, err := service.NewAppInstall().SearchForWebsite(request.AppInstalledSearch{
		PageInfo: dto.PageInfo{Page: 1, Limit: 200},
	})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	items := make([]mobileAppSummary, 0, len(apps))
	for _, app := range apps {
		items = append(items, mobileAppSummary{
			ID: app.ID, Name: app.Name, Version: app.Version, Status: app.Status,
			Description: app.Description, HTTPPort: app.HttpPort, HTTPSPort: app.HttpsPort,
			RuntimeHost: app.RuntimeHost, RuntimeKind: app.RuntimeKind,
		})
	}
	return c.JSON(e.Succ(fiber.Map{"items": items, "total": total}))
}
