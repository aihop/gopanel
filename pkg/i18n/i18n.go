package i18n

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3/log"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/gofiber/fiber/v3"
)

const (
	localsKey = "i18n"
	// LangKey 是 Fiber locals 中存放客户端语言的 key。
	LangKey = "i18n.lang"
)

var (
	defaultCfgMu sync.RWMutex
	defaultCfg   *Config
)

// New creates a new middleware handler
func New(config ...*Config) fiber.Handler {
	cfg := configDefault(config...)
	// init bundle
	bundle := i18n.NewBundle(cfg.DefaultLanguage)
	bundle.RegisterUnmarshalFunc(cfg.FormatBundleFile, cfg.UnmarshalFunc)
	cfg.bundle = bundle

	cfg.loadMessages()
	cfg.initLocalizerMap()

	// set package default config for no-ctx usage
	defaultCfgMu.Lock()
	defaultCfg = cfg
	defaultCfgMu.Unlock()

	return func(c fiber.Ctx) error {
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}
		c.Locals(localsKey, cfg)
		lang := cfg.LangHandler(c, cfg.DefaultLanguage.String())
		if lang == "" {
			lang = cfg.DefaultLanguage.String()
		}
		c.Locals(LangKey, lang)
		return c.Next()
	}
}

// GetLang 从 fiber.Ctx 中取出由本中间件写入的语言；ctx 为 nil 时返回空字符串。
func GetLang(c fiber.Ctx) string {
	if c == nil {
		return ""
	}
	v := c.Locals(LangKey)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getDefaultConfig() *Config {
	defaultCfgMu.RLock()
	defer defaultCfgMu.RUnlock()
	return defaultCfg
}

func (c *Config) loadMessage(filepath string) {
	buf, err := c.Loader.LoadMessage(filepath)
	if err != nil {
		log.Fatalf("i18n load message error: %v", err)
	}
	if _, err := c.bundle.ParseMessageFileBytes(buf, filepath); err != nil {
		log.Fatalf("i18n parse message error: %v", err)
	}
}

func (c *Config) loadMessages() *Config {
	for _, lang := range c.AcceptLanguages {
		bundleFilePath := fmt.Sprintf("%s.%s", lang.String(), c.FormatBundleFile)
		filepath := path.Join(c.RootPath, bundleFilePath)
		c.loadMessage(filepath)
	}
	return c
}

func (c *Config) initLocalizerMap() {
	localizerMap := &sync.Map{}

	defaultLang := c.DefaultLanguage.String()
	for _, lang := range c.AcceptLanguages {
		s := lang.String()
		// 构建回退链：当前语言命中失败时降级到默认语言（中文）
		if s == defaultLang {
			localizerMap.Store(s, i18n.NewLocalizer(c.bundle, s))
		} else {
			localizerMap.Store(s, i18n.NewLocalizer(c.bundle, s, defaultLang))
		}
	}

	if _, ok := localizerMap.Load(defaultLang); !ok {
		localizerMap.Store(defaultLang, i18n.NewLocalizer(c.bundle, defaultLang))
	}
	c.mu.Lock()
	c.localizerMap = localizerMap
	c.mu.Unlock()
}

func MustLocalize(params interface{}, lang string) string {
	message, err := Localize(params, lang)
	if err != nil {
		return ""
	}
	return message
}

func Localize(params interface{}, lang string) (string, error) {
	cfg := getDefaultConfig()
	if cfg == nil {
		return "", fmt.Errorf("i18n.Localize error: %v", "Config is nil (call i18n.New or set default config)")
	}

	// choose lang
	if strings.TrimSpace(lang) == "" {
		lang = cfg.DefaultLanguage.String()
	}

	localizerIface, _ := cfg.localizerMap.Load(lang)
	if localizerIface == nil {
		defaultLang := cfg.DefaultLanguage.String()
		localizerIface, _ = cfg.localizerMap.Load(defaultLang)
	}
	if localizerIface == nil {
		return "", fmt.Errorf("localizer is nil after attempting default language")
	}
	localizer := localizerIface.(*i18n.Localizer)
	var localizeConfig *i18n.LocalizeConfig
	switch paramValue := params.(type) {
	case string:
		localizeConfig = &i18n.LocalizeConfig{MessageID: paramValue}
	case *i18n.LocalizeConfig:
		localizeConfig = paramValue
	default:
		return "", fmt.Errorf("unsupported params type: %T", params)
	}
	message, err := localizer.Localize(localizeConfig)
	if err != nil {
		// go-i18n 在 fallback 命中时会同时返回 message 和 MessageNotFoundErr，
		// 此时 message 已可用，不应再冒泡错误。
		if message != "" {
			return message, nil
		}
		log.Errorf("i18n.Localize error: %v", err)
		return "", fmt.Errorf("i18n.Localize error: %v", err)
	}
	return message, nil
}
