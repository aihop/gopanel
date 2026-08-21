package i18n

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/text/language"
)

// defaultTestLang 让测试不依赖 ConfigDefault.DefaultLanguage（默认是 en）。
var defaultTestLang = language.Chinese

func TestNormalizeLangTag(t *testing.T) {
	cases := map[string]string{
		"":                 "",
		"en":               "en",
		"EN":               "en",
		"en-US":            "en-us",
		"en-US,en;q=0.9":   "en-us",
		" zh-CN ,zh;q=0.8": "zh-cn",
	}
	for in, want := range cases {
		got := normalizeLangTag(in)
		if got != want {
			t.Errorf("normalizeLangTag(%q) = %q, want %q", in, got, want)
		}
	}
}

// newMiddlewareTestApp 用一个会回显语言的下游 handler 验证中间件是否把 lang 写入 locals。
func newMiddlewareTestApp(t *testing.T) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(New(&Config{
		RootPath:         "",
		DefaultLanguage:  defaultTestLang,
		AcceptLanguages:  []language.Tag{defaultTestLang, language.English},
		FormatBundleFile: "yaml",
		UnmarshalFunc:    ConfigDefault.UnmarshalFunc,
		Loader:           LoaderFunc(func(string) ([]byte, error) { return nil, nil }),
		LangHandler:      defaultLangHandler,
	}))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString(GetLang(c))
	})
	return app
}

func TestMiddleware_DefaultLangHandler(t *testing.T) {
	app := newMiddlewareTestApp(t)
	cases := []struct {
		name       string
		acceptLang string
		queryLang  string
		want       string
	}{
		{"empty -> default (zh)", "", "", defaultTestLang.String()},
		{"query lang=zh overrides Accept-Language", "en", "zh", "zh"},
		{"Accept-Language en", "en", "", "en"},
		{"Accept-Language zh-CN", "zh-CN,zh;q=0.9", "", "zh"},
		{"Accept-Language ja -> default", "ja", "", defaultTestLang.String()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?lang="+tc.queryLang, nil)
			if tc.acceptLang != "" {
				req.Header.Set("Accept-Language", tc.acceptLang)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			buf, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			body := string(buf)
			if body != tc.want {
				t.Errorf("lang = %q, want %q", body, tc.want)
			}
		})
	}
}

func TestGetLang_NilCtx(t *testing.T) {
	if got := GetLang(nil); got != "" {
		t.Errorf("GetLang(nil) = %q, want empty", got)
	}
}
