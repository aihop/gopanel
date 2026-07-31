package api

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/buserr"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/init/geo"
	"github.com/aihop/gopanel/utils/cryptx"
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
	var req struct {
		DeviceTTLDays int `json:"deviceTtlDays"`
	}
	if len(c.Body()) > 0 {
		if err := c.Bind().JSON(&req); err != nil {
			return c.JSON(e.Fail(err))
		}
	}
	if req.DeviceTTLDays == 0 {
		req.DeviceTTLDays = service.DefaultMobileDeviceTTLDays
	}
	code, expiresAt, err := service.IssueMobilePairing(claims.UserId, req.DeviceTTLDays)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"code": code, "expiresAt": expiresAt, "deviceTtlDays": req.DeviceTTLDays}))
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
	setMobileDeviceCookie(c, deviceToken, device.ExpiresAt)
	return c.JSON(e.Succ(fiber.Map{"device": device}))
}

func LoginMobileDevice(c fiber.Ctx) error {
	var req struct {
		Email        string `json:"email"`
		Mobile       string `json:"mobile"`
		Password     string `json:"password"`
		CaptchaToken string `json:"captchaToken"`
		DeviceName   string `json:"deviceName"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	authReq := &dto.AuthSignin{
		Email: strings.TrimSpace(req.Email), Mobile: strings.TrimSpace(req.Mobile),
		Password: req.Password, CaptchaToken: req.CaptchaToken,
	}
	loginLog := model.LoginLog{
		IP: c.IP(), Agent: string(c.Request().Header.UserAgent()), Address: geo.Region(c.IP()),
	}
	logService := service.NewLogService()
	if message, blocked := defaultLoginAttemptGuard.Check(c.IP(), authReq); blocked {
		loginLog.Status = constant.StatusFailed
		loginLog.Message = "mobile login blocked by rate limiter"
		_ = logService.CreateLoginLog(loginLog)
		return c.JSON(e.RetError(constant.StatusCodeFullFail, message))
	}
	if defaultLoginAttemptGuard.RequiresCaptcha(c.IP(), authReq) {
		if err := consumeVerifiedLoginCaptchaToken(authReq.CaptchaToken); err != nil {
			loginLog.Status = constant.StatusFailed
			loginLog.Message = "mobile login captcha verification required"
			_ = logService.CreateLoginLog(loginLog)
			return c.JSON(e.RetError(constant.StatusCodeFullFail, "请先完成滑块验证"))
		}
	}

	failLogin := func(logMessage string) error {
		loginLog.Status = constant.StatusFailed
		loginLog.Message = logMessage
		defaultLoginAttemptGuard.RegisterFailure(c.IP(), authReq)
		time.Sleep(loginAttemptDelay)
		_ = logService.CreateLoginLog(loginLog)
		return c.JSON(e.Fail(buserr.WithDetail("ErrLoginFailed", logMessage)))
	}
	userService := service.NewUser()
	var user *model.User
	var err error
	if authReq.Email != "" {
		user, err = userService.GetByEmail(authReq.Email)
	} else if authReq.Mobile != "" {
		user, err = userService.GetByMobile(authReq.Mobile)
	}
	if err != nil || user == nil {
		return failLogin("user not found")
	}
	if !canLoginMobileConsole(user) {
		return failLogin("user is not an active administrator")
	}
	if !cryptx.ValidatePassword(user.Password, authReq.Password) {
		return failLogin("password verification failed")
	}
	deviceToken, device, err := service.AuthorizeMobileDevice(
		user.ID, req.DeviceName, c.IP(), string(c.Request().Header.UserAgent()), service.DefaultMobileDeviceTTLDays,
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	defaultLoginAttemptGuard.RegisterSuccess(c.IP(), authReq)
	loginLog.Status = constant.StatusSuccess
	loginLog.Message = "Mobile login success"
	_ = logService.CreateLoginLog(loginLog)
	_ = userService.Update(&model.User{ID: user.ID, LoginAt: time.Now()})
	setMobileDeviceCookie(c, deviceToken, device.ExpiresAt)
	return c.JSON(e.Succ(fiber.Map{"device": device}))
}

func canLoginMobileConsole(user *model.User) bool {
	if user == nil || user.Status != constant.UserStatusNormal {
		return false
	}
	return user.Role == constant.UserRoleAdmin || user.Role == constant.UserRoleSuper
}

func setMobileDeviceCookie(c fiber.Ctx, deviceToken string, expiresAt time.Time) {
	c.Cookie(&fiber.Cookie{
		Name: "gopanel_mobile", Value: deviceToken, Path: "/api/mobile/app", Expires: expiresAt,
		HTTPOnly: true, SameSite: "Lax", Secure: c.Scheme() == "https",
	})
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

func GetMobileNodes(c fiber.Ctx) error {
	nodes, err := service.NewNode().MobileList()
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(nodes))
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
