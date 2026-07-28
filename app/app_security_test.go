package app

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/middleware"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/pprof"
)

func TestPprofRequiresAuthentication(t *testing.T) {
	old := global.CONF
	global.CONF.System.Mode = "prod"
	global.CONF.System.Entrance = ""
	global.CONF.System.ApiInterfaceStatus = "Closed"
	t.Cleanup(func() { global.CONF = old })

	server := fiber.New()
	server.Use(middleware.Entrance)
	server.Use("/debug/pprof", middleware.JWT(constant.UserRoleSuper))
	server.Use(pprof.New())
	resp, err := server.Test(httptest.NewRequest("GET", "/debug/pprof/", nil))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "Types of profiles available") {
		t.Fatal("pprof response was exposed without authentication")
	}
}
