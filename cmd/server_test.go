package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/glebarez/sqlite"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func TestValidateNewPassword(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "空", input: "", wantErr: true},
		{name: "只有空格", input: "   ", wantErr: true},
		{name: "太短", input: "abc", wantErr: true},
		{name: "刚好6位", input: "abc123", want: "abc123"},
		// fmt.Scan 时代含空格的密码会被截断成第一个 token，这里必须原样保留
		{name: "含空格", input: "My Pass 123", want: "My Pass 123"},
		{name: "超过72字节", input: strings.Repeat("a", 73), wantErr: true},
		{name: "刚好72字节", input: strings.Repeat("a", 72), want: strings.Repeat("a", 72)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateNewPassword(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际通过并返回 %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("不该报错: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolvePanelDBFile(t *testing.T) {
	// 环境变量优先级最高（与 init/conf 的解析顺序一致）
	t.Setenv("GOPANEL_BASE_DIR", "/env/base")
	viper.Set("system.base_dir", "/conf/base")
	if got, want := resolvePanelDBFile(), filepath.Join("/env/base", "db", "gopanel.db"); got != want {
		t.Fatalf("env 优先级错误: got %q, want %q", got, want)
	}

	// 没有环境变量时用 conf 里的 system.base_dir
	t.Setenv("GOPANEL_BASE_DIR", "")
	t.Setenv("GPC_BASE_DIR", "")
	if got, want := resolvePanelDBFile(), filepath.Join("/conf/base", "db", "gopanel.db"); got != want {
		t.Fatalf("conf base_dir 解析错误: got %q, want %q", got, want)
	}
	viper.Set("system.base_dir", "")
}

func newTestUserDB(t *testing.T, users []model.User) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "gopanel.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Table("user").AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for i := range users {
		if err := db.Table("user").Create(&users[i]).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	return db
}

func TestPickUserToReset(t *testing.T) {
	users := []model.User{
		{Email: "super@a.com", Role: "SUPER", Status: 20},
		{Email: "sub@a.com", Role: "SUB_ADMIN", Status: 20},
		{Email: "off@a.com", Role: "USER", Status: 10},
	}

	t.Run("不给邮箱时取超管", func(t *testing.T) {
		db := newTestUserDB(t, users)
		got, err := pickUserToReset(db, "")
		if err != nil {
			t.Fatalf("不该报错: %v", err)
		}
		if got.Email != "super@a.com" {
			t.Fatalf("got %q", got.Email)
		}
	})

	t.Run("按邮箱定位非超管账号", func(t *testing.T) {
		db := newTestUserDB(t, users)
		got, err := pickUserToReset(db, "sub@a.com")
		if err != nil {
			t.Fatalf("不该报错: %v", err)
		}
		if got.Role != "SUB_ADMIN" {
			t.Fatalf("got role %q", got.Role)
		}
	})

	// 原实现写死了 status = 20，超管被停用后直接"查不到用户"，无法自救
	t.Run("停用状态的账号也能定位", func(t *testing.T) {
		db := newTestUserDB(t, []model.User{{Email: "super@a.com", Role: "SUPER", Status: 10}})
		got, err := pickUserToReset(db, "")
		if err != nil {
			t.Fatalf("不该报错: %v", err)
		}
		if got.Status == 20 {
			t.Fatalf("测试数据没生效，status=%d", got.Status)
		}
	})

	t.Run("多个超管时要求指定邮箱并列出候选", func(t *testing.T) {
		db := newTestUserDB(t, []model.User{
			{Email: "s1@a.com", Role: "SUPER", Status: 20},
			{Email: "s2@a.com", Role: "SUPER", Status: 20},
		})
		_, err := pickUserToReset(db, "")
		if err == nil {
			t.Fatal("期望报错")
		}
		for _, want := range []string{"s1@a.com", "s2@a.com"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("错误信息里缺少 %q: %v", want, err)
			}
		}
	})

	t.Run("邮箱不存在时列出现有账号", func(t *testing.T) {
		db := newTestUserDB(t, users)
		_, err := pickUserToReset(db, "nobody@a.com")
		if err == nil {
			t.Fatal("期望报错")
		}
		if !strings.Contains(err.Error(), "super@a.com") {
			t.Fatalf("错误信息里没有列出现有账号: %v", err)
		}
	})

	t.Run("空库", func(t *testing.T) {
		db := newTestUserDB(t, nil)
		_, err := pickUserToReset(db, "")
		if err == nil {
			t.Fatal("期望报错")
		}
		if !strings.Contains(err.Error(), "没有任何账号") {
			t.Fatalf("提示不清晰: %v", err)
		}
	})
}
