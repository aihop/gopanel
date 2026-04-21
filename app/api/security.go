package api

import (
	"context"
	"encoding/json"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/utils/gpc"
	"github.com/gofiber/fiber/v3"
)

func SecurityScan(c fiber.Ctx) error {
	// 1. Scan SSH config
	sshResp, err := gpc.Do(context.Background(), "SECURITY_SCAN_SSH", nil)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var sshConfig map[string]interface{}
	if err := json.Unmarshal([]byte(sshResp.Output), &sshConfig); err != nil {
		return c.JSON(e.Fail(err))
	}

	// 2. Scan high-risk exposed ports
	portResp, err := gpc.Do(context.Background(), "SECURITY_SCAN_PORT", nil)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var portConfig map[string]interface{}
	if err := json.Unmarshal([]byte(portResp.Output), &portConfig); err != nil {
		return c.JSON(e.Fail(err))
	}

	// Combine results
	result := map[string]interface{}{
		"ssh":  sshConfig,
		"port": portConfig,
	}

	return c.JSON(e.Succ(result))
}

func SecurityFixSSH(c fiber.Ctx) error {
	resp, err := gpc.Do(context.Background(), "SECURITY_FIX_SSH", nil)
	if err != nil {
		return c.JSON(e.Fail(err))
	}

	var fixResult map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Output), &fixResult); err != nil {
		return c.JSON(e.Fail(err))
	}

	return c.JSON(e.Succ(fixResult))
}