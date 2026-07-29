package api

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type mobilePairingAttempt struct {
	WindowStarted time.Time
	Count         int
}

var mobilePairingAttempts = struct {
	sync.Mutex
	items       map[string]mobilePairingAttempt
	lastCleanup time.Time
}{items: make(map[string]mobilePairingAttempt)}

const maxMobilePairingAttemptEntries = 4096

func IssueMobilePairing(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	code, expiresAt, err := service.IssueMobilePairing(claims.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"code": code, "expiresAt": expiresAt}))
}

func ExchangeMobilePairing(c fiber.Ctx) error {
	if !allowMobilePairingAttempt(c.IP(), time.Now()) {
		return c.JSON(e.Fail(errors.New("配对尝试过于频繁，请稍后再试")))
	}
	var req struct {
		Code       string `json:"code"`
		DeviceName string `json:"deviceName"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	deviceToken, device, err := service.ExchangeMobilePairing(
		req.Code, req.DeviceName, c.IP(), string(c.Request().Header.UserAgent()),
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	c.Cookie(&fiber.Cookie{
		Name:     "gopanel_mobile",
		Value:    deviceToken,
		Path:     "/api/mobile/app",
		Expires:  device.ExpiresAt,
		HTTPOnly: true,
		SameSite: "Lax",
		Secure:   c.Scheme() == "https",
	})
	return c.JSON(e.Succ(fiber.Map{"device": device}))
}

func allowMobilePairingAttempt(ip string, now time.Time) bool {
	mobilePairingAttempts.Lock()
	defer mobilePairingAttempts.Unlock()
	if mobilePairingAttempts.lastCleanup.IsZero() || now.Sub(mobilePairingAttempts.lastCleanup) >= time.Minute {
		for storedIP, attempt := range mobilePairingAttempts.items {
			if now.Sub(attempt.WindowStarted) >= time.Minute {
				delete(mobilePairingAttempts.items, storedIP)
			}
		}
		mobilePairingAttempts.lastCleanup = now
	}
	attempt, exists := mobilePairingAttempts.items[ip]
	if !exists && len(mobilePairingAttempts.items) >= maxMobilePairingAttemptEntries {
		return false
	}
	if attempt.WindowStarted.IsZero() || now.Sub(attempt.WindowStarted) >= time.Minute {
		mobilePairingAttempts.items[ip] = mobilePairingAttempt{WindowStarted: now, Count: 1}
		return true
	}
	if attempt.Count >= 20 {
		return false
	}
	attempt.Count++
	mobilePairingAttempts.items[ip] = attempt
	return true
}

func ListMobileDevices(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	devices, err := repo.NewMobileAccessRepo().ListDevices(claims.UserId)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"items": devices, "total": len(devices)}))
}

func RevokeMobileDevice(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	deviceID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || deviceID == 0 {
		return c.JSON(e.Fail(errors.New("手机设备参数无效")))
	}
	if err := repo.NewMobileAccessRepo().RevokeDevice(uint(deviceID), claims.UserId); err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ())
}

func GetMobileOverview(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	current := dashboardService.LoadCurrentInfo(dto.DashboardReq{Scope: "basic"})
	sessions, total, err := repo.NewAIDevSessionRepo().GetSessionsByUserID(claims.UserId, 0, 1, 20)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	approvals, err := repo.NewAIDevSessionRepo().GetApprovalsByUserID(claims.UserId, "pending", 20)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"system":           current,
		"sessions":         sessions,
		"sessionTotal":     total,
		"pendingApprovals": approvals,
		"serverTime":       time.Now(),
	}))
}

func LogoutMobileDevice(c fiber.Ctx) error {
	deviceID, _ := c.Locals("mobile_device_id").(uint)
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if deviceID > 0 {
		_ = repo.NewMobileAccessRepo().RevokeDevice(deviceID, claims.UserId)
	}
	c.Cookie(&fiber.Cookie{
		Name:     "gopanel_mobile",
		Value:    "",
		Path:     "/api/mobile/app",
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		SameSite: "Lax",
	})
	return c.JSON(e.Succ())
}

func MobileHealth(c fiber.Ctx) error {
	return c.JSON(e.Succ(fiber.Map{"available": true, "scheme": strings.ToLower(c.Scheme())}))
}
