package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/aihop/gopanel/app/dto"
	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cryptx"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/wenlng/go-captcha-assets/resources/imagesv2"
	"github.com/wenlng/go-captcha-assets/resources/tiles"
	"github.com/wenlng/go-captcha/v2/base/option"
	"github.com/wenlng/go-captcha/v2/slide"
)

const (
	loginCaptchaChallengeTTL = 2 * time.Minute
	loginCaptchaVerifyTTL    = 10 * time.Minute
	loginCaptchaVisitorTTL   = 24 * time.Hour
	loginCaptchaTolerance    = 6.0
)

type loginCaptchaChallenge struct {
	TargetX   float64 `json:"targetX"`
	SecretKey string  `json:"secretKey"`
}

type loginCaptchaPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

var (
	loginSlideCaptcha     slide.Captcha
	loginSlideCaptchaErr  error
	loginSlideCaptchaOnce sync.Once
)

func VerifyCaptchaGet(c fiber.Ctx) error {
	_, _ = e.BodyToStruct[dto.VerifyCaptchaGetReq](c.Body())

	captchaEngine, err := getLoginSlideCaptcha()
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	captchaData, err := captchaEngine.Generate()
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	masterImage, err := captchaData.GetMasterImage().ToBase64()
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	tileImage, err := captchaData.GetTileImage().ToBase64()
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	token := uuid.NewString()
	secretKey := buildCaptchaSecretKey()
	challenge := loginCaptchaChallenge{
		TargetX:   float64(captchaData.GetData().X),
		SecretKey: secretKey,
	}
	if err := saveLoginCaptchaChallenge(token, challenge); err != nil {
		return c.JSON(e.Fail(err))
	}

	isOK := isTrustedCaptchaVisitor(c)
	if isOK {
		if err := storeVerifiedLoginCaptchaToken(token, loginCaptchaVerifyTTL); err != nil {
			return c.JSON(e.Fail(err))
		}
	}

	return c.JSON(e.Succ(&dto.VerifyCaptchaGetResp{
		OriginalImg: masterImage,
		BlockImg:    tileImage,
		Token:       token,
		SecretKey:   secretKey,
		IsOK:        isOK,
		PieceTop:    captchaData.GetData().Y,
		PieceWidth:  captchaData.GetData().Width,
		PieceHeight: captchaData.GetData().Height,
	}))
}

func VerifyCaptchaCheck(c fiber.Ctx) error {
	req, err := e.BodyToStruct[dto.VerifyCaptchaCheckReq](c.Body())
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	challenge, err := getLoginCaptchaChallenge(req.Token)
	if err != nil {
		return c.JSON(e.RetError(constant.StatusCodeFullFail, "validation failed: already expired"))
	}

	pointRaw, err := cryptx.AesDecrypt(req.Point, challenge.SecretKey)
	if err != nil {
		return c.JSON(e.RetError(constant.StatusCodeFullFail, "point decrypt failed"))
	}

	var point loginCaptchaPoint
	if err := json.Unmarshal([]byte(pointRaw), &point); err != nil {
		return c.JSON(e.RetError(constant.StatusCodeFullFail, "point parse failed"))
	}

	if math.Abs(point.X-challenge.TargetX) > loginCaptchaTolerance {
		_ = deleteLoginCaptchaChallenge(req.Token)
		return c.JSON(e.RetError(constant.StatusCodeFullFail, "validation failed"))
	}

	if err := deleteLoginCaptchaChallenge(req.Token); err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := storeVerifiedLoginCaptchaToken(req.Token, loginCaptchaVerifyTTL); err != nil {
		return c.JSON(e.Fail(err))
	}
	if err := trustCaptchaVisitor(c); err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(&dto.Token{Token: req.Token}))
}

func consumeVerifiedLoginCaptchaToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New(constant.ErrVerifyToken)
	}
	key := loginCaptchaVerifyKey(token)
	if _, err := global.CACHE.Get(key); err != nil {
		return errors.New(constant.ErrVerifyToken)
	}
	return global.CACHE.Del(key)
}

func storeVerifiedLoginCaptchaToken(token string, ttl time.Duration) error {
	return global.CACHE.SetWithTTL(loginCaptchaVerifyKey(token), "1", ttl)
}

func saveLoginCaptchaChallenge(token string, challenge loginCaptchaChallenge) error {
	body, err := json.Marshal(challenge)
	if err != nil {
		return err
	}
	return global.CACHE.SetWithTTL(loginCaptchaChallengeKey(token), string(body), loginCaptchaChallengeTTL)
}

func getLoginCaptchaChallenge(token string) (*loginCaptchaChallenge, error) {
	body, err := global.CACHE.Get(loginCaptchaChallengeKey(token))
	if err != nil {
		return nil, err
	}
	var challenge loginCaptchaChallenge
	if err := json.Unmarshal(body, &challenge); err != nil {
		return nil, err
	}
	return &challenge, nil
}

func deleteLoginCaptchaChallenge(token string) error {
	return global.CACHE.Del(loginCaptchaChallengeKey(token))
}

func loginCaptchaChallengeKey(token string) string {
	return "auth:slide:challenge:" + strings.TrimSpace(token)
}

func loginCaptchaVerifyKey(token string) string {
	return "auth:slide:verified:" + strings.TrimSpace(token)
}

func loginCaptchaVisitorKey(c fiber.Ctx) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		strings.TrimSpace(c.IP()),
		strings.TrimSpace(c.Get("User-Agent")),
		strings.TrimSpace(c.Get("Accept-Language")),
	}, "|")))
	return "auth:slide:visitor:" + hex.EncodeToString(sum[:])
}

func isTrustedCaptchaVisitor(c fiber.Ctx) bool {
	_, err := global.CACHE.Get(loginCaptchaVisitorKey(c))
	return err == nil
}

func trustCaptchaVisitor(c fiber.Ctx) error {
	return global.CACHE.SetWithTTL(loginCaptchaVisitorKey(c), "1", loginCaptchaVisitorTTL)
}

func buildCaptchaSecretKey() string {
	raw := strings.ReplaceAll(uuid.NewString(), "-", "")
	return raw[:16]
}

func getLoginSlideCaptcha() (slide.Captcha, error) {
	loginSlideCaptchaOnce.Do(func() {
		builder := slide.NewBuilder(
			slide.WithImageSize(option.Size{Width: 330, Height: 155}),
			slide.WithEnableGraphVerticalRandom(false),
		)

		backgrounds, err := imagesv2.GetImages()
		if err != nil {
			loginSlideCaptchaErr = err
			return
		}
		graphs, err := tiles.GetTiles()
		if err != nil {
			loginSlideCaptchaErr = err
			return
		}

		slideGraphs := make([]*slide.GraphImage, 0, len(graphs))
		for _, graph := range graphs {
			slideGraphs = append(slideGraphs, &slide.GraphImage{
				OverlayImage: graph.OverlayImage,
				MaskImage:    graph.MaskImage,
				ShadowImage:  graph.ShadowImage,
			})
		}

		builder.SetResources(
			slide.WithBackgrounds(backgrounds),
			slide.WithGraphImages(slideGraphs),
		)
		loginSlideCaptcha = builder.Make()
	})
	return loginSlideCaptcha, loginSlideCaptchaErr
}
