package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/encrypt"
)

func TestCodeGitCredentialEnvironmentUsesEncryptedTemporaryAskPass(t *testing.T) {
	database := withCodeGovernanceDB(t)
	if err := database.AutoMigrate(&model.AIGitCredential{}); err != nil {
		t.Fatal(err)
	}
	oldKey := global.CONF.System.EncryptKey
	global.CONF.System.EncryptKey = "0123456789abcdef0123456789abcdef"
	t.Cleanup(func() { global.CONF.System.EncryptKey = oldKey })
	ciphertext, err := encrypt.StringEncrypt("secret-token")
	if err != nil {
		t.Fatal(err)
	}
	credential := &model.AIGitCredential{CreatorID: 1, Name: "Codeup", Username: "git-user", Secret: ciphertext}
	if err := database.Create(credential).Error; err != nil {
		t.Fatal(err)
	}

	environment, cleanup, err := codeGitCredentialEnvironment(credential.ID, []string{"PATH=/usr/bin"})
	if err != nil {
		t.Fatal(err)
	}
	values := make(map[string]string)
	for _, item := range environment {
		key, value, found := strings.Cut(item, "=")
		if found {
			values[key] = value
		}
	}
	if values["GIT_ASKPASS_REQUIRE"] != "force" || values["GIT_ASKPASS"] == "" {
		t.Fatalf("AskPass environment is incomplete: %#v", values)
	}
	secretPath := values["GOPANEL_GIT_SECRET_FILE"]
	secret, err := os.ReadFile(secretPath)
	if err != nil || string(secret) != "secret-token" {
		t.Fatalf("temporary credential is unavailable: %q, %v", secret, err)
	}
	info, err := os.Stat(secretPath)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("temporary credential permissions = %v, %v", info, err)
	}
	cleanup()
	if _, err := os.Stat(filepath.Dir(secretPath)); !os.IsNotExist(err) {
		t.Fatalf("temporary credential directory was not removed: %v", err)
	}
}

func TestCodeDeliveryFailureCodeClassifiesRemoteFailures(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{errCodeGitAuthentication, "authentication_failed"},
		{errCodeGitRepository, "repository_unavailable"},
		{errCodeGitNetwork, "network_failed"},
	} {
		if got := codeDeliveryFailureCode(codeDeliveryStagePushing, test.err); got != test.want {
			t.Fatalf("failure code = %q, want %q", got, test.want)
		}
	}
}

func TestNormalizeCodeGitCommandErrorSeparatesRepositoryAndNetwork(t *testing.T) {
	repositoryErr := normalizeCodeGitCommandError("remote: unknown repository path\nfatal: repository not found")
	if !strings.Contains(repositoryErr.Error(), "仓库不存在") || strings.Contains(repositoryErr.Error(), "认证失败") {
		t.Fatalf("repository failure was misclassified: %v", repositoryErr)
	}
	networkErr := normalizeCodeGitCommandError("fatal: failed to connect to 127.0.0.1: connection refused")
	if !strings.Contains(networkErr.Error(), "网络不可用") || !strings.Contains(networkErr.Error(), "connection refused") {
		t.Fatalf("network failure lost classification or detail: %v", networkErr)
	}
}
