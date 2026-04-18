package service

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/cmd"
)

func handleUnTar(sourceFile, targetDir string, secret string) error {
	if _, err := os.Stat(targetDir); err != nil && os.IsNotExist(err) {
		if err = os.MkdirAll(targetDir, os.ModePerm); err != nil {
			return err
		}
	}
	commands := ""
	if len(secret) != 0 {
		extraCmd := "openssl enc -d -aes-256-cbc -k '" + secret + "' -in " + sourceFile + " | "
		commands = fmt.Sprintf("%s tar -zxvf - -C %s", extraCmd, targetDir+" > /dev/null 2>&1")
		global.LOG.Debug(strings.ReplaceAll(commands, fmt.Sprintf(" %s ", secret), "******"))
	} else {
		commands = fmt.Sprintf("tar zxvfC %s %s", sourceFile, targetDir)
		global.LOG.Debug(commands)
	}

	stdout, err := cmd.ExecWithTimeOut(commands, 24*time.Hour)
	if err != nil {
		global.LOG.Errorf("do handle untar failed, stdout: %s, err: %v", stdout, err)
		return errors.New(stdout)
	}
	return nil
}
