package api

import (
	"errors"
	"strconv"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func OpenCodeProjectTerminal(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || projectID == 0 {
		return c.JSON(e.Fail(errors.New("项目参数无效")))
	}
	project, err := getCodeProjectWithPermission(uint(projectID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	workDir, err := codeProjectTerminalWorkDir(project, claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if existing := findRunningCodeProjectTerminal(claims.UserId, workDir); existing != nil {
		return c.JSON(e.Succ(existing))
	}
	record, err := hostTerminals.create(createHostTerminalRequest{
		Shell: "default", WorkDir: workDir, Cols: 120, Rows: 32,
	}, claims.UserId, c.IP())
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(record))
}

func codeProjectTerminalWorkDir(project *model.AIProject, claims *token.CustomClaims) (string, error) {
	if claims == nil || (claims.Role != constant.UserRoleAdmin && claims.Role != constant.UserRoleSuper) {
		return "", errors.New("只有管理员可以使用项目原生终端")
	}
	return aiProjectSessionWorkDir(project, claims)
}

func findRunningCodeProjectTerminal(userID uint, workDir string) *model.HostTerminalSession {
	var records []model.HostTerminalSession
	if err := global.DB.Where(
		"user_id = ? AND work_dir = ? AND status IN ?", userID, workDir, []string{"starting", "running"},
	).Order("created_at desc").Find(&records).Error; err != nil {
		return nil
	}
	for index := range records {
		resumed, err := hostTerminals.resume(records[index].ID)
		if err == nil {
			return resumed
		}
		markHostTerminalInterrupted(&records[index])
	}
	return nil
}
