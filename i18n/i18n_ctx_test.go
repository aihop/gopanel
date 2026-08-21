package i18n

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"

	paneli18n "github.com/aihop/gopanel/pkg/i18n"
)

// testMessagesInMemory 返回按文件路径分发测试 yaml 的 Loader。
// path 由 pkg/i18n 拼接为 "<root>/<lang>.yaml"。
func testMessagesInMemory(zh, en string) fiber.Handler {
	bundle := make(map[string][]byte)
	bundle["zh.yaml"] = []byte(zh)
	bundle["en.yaml"] = []byte(en)
	return paneli18n.New(&paneli18n.Config{
		RootPath:         "",
		DefaultLanguage:  language.Chinese,
		AcceptLanguages:  []language.Tag{language.Chinese, language.English},
		FormatBundleFile: "yaml",
		UnmarshalFunc:    yaml.Unmarshal,
		Loader: paneli18n.LoaderFunc(func(path string) ([]byte, error) {
			base := path[strings.LastIndex(path, "/")+1:]
			if v, ok := bundle[base]; ok {
				return v, nil
			}
			return nil, nil
		}),
		LangHandler: defaultLangHandlerForTest,
	})
}

// defaultLangHandlerForTest 直接复用 pkg/i18n 的 defaultLangHandler；通过函数变量引用避免循环。
var defaultLangHandlerForTest = paneli18n.DefaultLangHandler

// yamlUnmarshal 引用 yaml.v3 的 Unmarshal 函数。
var yamlUnmarshal = yaml.Unmarshal

// 实际测试时通过 TestMain 注入测试 fixture。

// newWrapperTestApp 构造注入测试 key 的 fiber.App，下游回显 GetMsgFromCtx 结果。
func newWrapperTestApp(t *testing.T) *fiber.App {
	t.Helper()
	const zh = `
CtxHello: "你好"
CtxGreet: "你好，{{ .name }}"
OnlyZhKey: "仅中文"
`
	const en = `
CtxHello: "Hello"
CtxGreet: "Hi, {{ .name }}"
`
	app := fiber.New()
	app.Use(testMessagesInMemory(zh, en))
	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString(GetMsgFromCtx(c, "CtxHello"))
	})
	app.Get("/greet/:name", func(c fiber.Ctx) error {
		return c.SendString(GetMsgFromCtx(c, "CtxGreet", map[string]interface{}{"name": c.Params("name")}))
	})
	app.Get("/fallback", func(c fiber.Ctx) error {
		return c.SendString(GetMsgByLang("en", "OnlyZhKey"))
	})
	return app
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func TestGetMsgFromCtx_PicksLangFromAcceptLanguage(t *testing.T) {
	app := newWrapperTestApp(t)
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "en")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if got := readBody(t, resp); got != "Hello" {
		t.Errorf("expected Hello, got %q", got)
	}
}

func TestGetMsgFromCtx_DefaultZh(t *testing.T) {
	app := newWrapperTestApp(t)
	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if got := readBody(t, resp); got != "你好" {
		t.Errorf("expected 你好, got %q", got)
	}
}

func TestGetMsgFromCtx_WithTemplateData(t *testing.T) {
	app := newWrapperTestApp(t)
	req := httptest.NewRequest("GET", "/greet/张三", nil)
	req.Header.Set("Accept-Language", "en")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if got := readBody(t, resp); got != "Hi, 张三" {
		t.Errorf("expected 'Hi, 张三', got %q", got)
	}
}

func TestGetMsgByLang_FallbackToDefault(t *testing.T) {
	app := newWrapperTestApp(t)
	req := httptest.NewRequest("GET", "/fallback", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if got := readBody(t, resp); got != "仅中文" {
		t.Errorf("expected '仅中文' fallback for missing en key, got %q", got)
	}
}

func TestGetMsgFromCtx_NilCtxReturnsMissingKey(t *testing.T) {
	if got := GetMsgFromCtx(nil, "DefinitelyMissingKeyForTest_42"); got != "DefinitelyMissingKeyForTest_42" {
		t.Errorf("expected missing key fallback, got %q", got)
	}
}

func TestGetMsgByLang_MissingKeyReturnsKey(t *testing.T) {
	if got := GetMsgByLang("en", "DefinitelyMissingKeyForTest_42"); got != "DefinitelyMissingKeyForTest_42" {
		t.Errorf("expected missing key to be returned, got %q", got)
	}
}
