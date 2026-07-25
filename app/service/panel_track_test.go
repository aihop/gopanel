package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/pkg/zlog"
)

func TestTrackEventSmoke(t *testing.T) {
	global.LOG = zlog.New(io.Discard, zlog.DebugLevel, &zlog.TextFormatter{})
	baseDir := t.TempDir()
	global.CONF.System.BaseDir = baseDir
	installIDCache = ""

	got := make(chan url.Values, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got <- r.URL.Query()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	old := constant.TrackUrl
	constant.TrackUrl = srv.URL + "/api/panel/installs/track"
	defer func() { constant.TrackUrl = old }()

	TrackEvent(TrackEventUpgradeSuccess, "1.2.2")

	q := <-got
	t.Logf("query: %v", q)
	for _, key := range []string{"event", "install_id", "channel", "os", "arch", "source", "version"} {
		if q.Get(key) == "" {
			t.Fatalf("missing %s in %v", key, q)
		}
	}
	if q.Get("event") != "upgrade_success" || q.Get("version") != "1.2.2" || q.Get("source") != "panel" {
		t.Fatalf("unexpected query: %v", q)
	}

	// install_id 落盘并可复用
	content, err := os.ReadFile(filepath.Join(baseDir, "install_id"))
	if err != nil {
		t.Fatalf("install_id not persisted: %v", err)
	}
	if string(content) != q.Get("install_id") {
		t.Fatalf("install_id mismatch: file=%s query=%s", content, q.Get("install_id"))
	}
	installIDCache = ""
	if InstallID() != string(content) {
		t.Fatalf("install_id not reused from disk")
	}
}
