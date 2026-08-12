package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/encrypt"
)

func codeProjectGitCredentialID(projectID uint) uint {
	if projectID == 0 || global.DB == nil {
		return 0
	}
	var credentialID uint
	_ = global.DB.Model(&model.AIProject{}).Where("id = ?", projectID).Pluck("git_credential_id", &credentialID).Error
	return credentialID
}

func codeGitCredentialEnvironment(credentialID uint, base []string) ([]string, func(), error) {
	if credentialID == 0 {
		return base, func() {}, nil
	}
	var credential model.AIGitCredential
	if err := global.DB.First(&credential, credentialID).Error; err != nil {
		return nil, nil, errors.New("项目绑定的 Git 凭据不存在，请重新选择凭据")
	}
	secret, err := encrypt.StringDecrypt(credential.Secret)
	if err != nil {
		return nil, nil, errors.New("项目绑定的 Git 凭据无法解密，请重新保存凭据")
	}
	return codeGitCredentialEnvironmentFor(credential.Username, secret, base)
}

// codeGitCredentialEnvironmentFor 用给定的用户名和明文密钥搭出 Git 凭据环境。
//
// 与按 ID 取库的版本分开，是为了能校验「还没入库的凭据」——
// 保存时要验的是用户刚填的这一份，不是库里那份旧的。
func codeGitCredentialEnvironmentFor(username, secret string, base []string) ([]string, func(), error) {
	tempDir, err := os.MkdirTemp("", "gopanel-git-credential-")
	if err != nil {
		return nil, nil, fmt.Errorf("准备 Git 凭据失败：%w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }
	usernameFile := filepath.Join(tempDir, "username")
	secretFile := filepath.Join(tempDir, "secret")
	if err := os.WriteFile(usernameFile, []byte(username), 0600); err != nil {
		cleanup()
		return nil, nil, err
	}
	if err := os.WriteFile(secretFile, []byte(secret), 0600); err != nil {
		cleanup()
		return nil, nil, err
	}
	askPass, err := writeCodeGitAskPass(tempDir)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	env := upsertEnvironment(base, "GIT_ASKPASS", askPass)
	env = upsertEnvironment(env, "GIT_ASKPASS_REQUIRE", "force")
	env = upsertEnvironment(env, "GOPANEL_GIT_USERNAME_FILE", usernameFile)
	env = upsertEnvironment(env, "GOPANEL_GIT_SECRET_FILE", secretFile)
	return env, cleanup, nil
}

func writeCodeGitAskPass(tempDir string) (string, error) {
	path := filepath.Join(tempDir, "askpass.sh")
	content := "#!/bin/sh\ncase \"$1\" in\n  *sername*) cat \"$GOPANEL_GIT_USERNAME_FILE\" ;;\n  *) cat \"$GOPANEL_GIT_SECRET_FILE\" ;;\nesac\n"
	if runtime.GOOS == "windows" {
		path = filepath.Join(tempDir, "askpass.bat")
		content = "@echo off\necho %~1 | findstr /I username >nul\nif %errorlevel%==0 (type \"%GOPANEL_GIT_USERNAME_FILE%\") else (type \"%GOPANEL_GIT_SECRET_FILE%\")\n"
	}
	if err := os.WriteFile(path, []byte(content), 0700); err != nil {
		return "", err
	}
	return path, nil
}

func runCodeGitWithCredential(workDir string, timeout time.Duration, credentialID uint, args ...string) (string, error) {
	return runCodeGitCommand(workDir, timeout, credentialID, args...)
}

func parseCodeGitCredentialID(value string) (uint, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	return uint(id), err
}
