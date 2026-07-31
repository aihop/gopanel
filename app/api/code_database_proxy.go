package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/app/repo"
	"github.com/aihop/gopanel/app/service"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/global"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type codeDatabaseAccessView struct {
	ID           uint               `json:"id"`
	ProjectID    uint               `json:"projectId"`
	Alias        string             `json:"alias"`
	DatabaseName string             `json:"databaseName"`
	ServerName   string             `json:"serverName"`
	ServerType   model.DatabaseType `json:"serverType"`
	ReadOnly     bool               `json:"readOnly"`
}

func getCodeProjectWithPermission(projectID uint, claims *token.CustomClaims) (*model.AIGroup, error) {
	project, err := repo.NewAIGroupRepo().GetGroupByID(projectID)
	if err != nil {
		return nil, errors.New("项目不存在")
	}
	if !canManageAIProject(project, claims) {
		return nil, errors.New("无权访问该项目")
	}
	return project, nil
}

func listCodeDatabaseAccesses(projectID uint) ([]codeDatabaseAccessView, error) {
	var accesses []model.AICodeDatabaseAccess
	if err := global.DB.Preload("Server").Where("project_id = ?", projectID).Order("id asc").Find(&accesses).Error; err != nil {
		return nil, err
	}
	views := make([]codeDatabaseAccessView, 0, len(accesses))
	for _, access := range accesses {
		view := codeDatabaseAccessView{ID: access.ID, ProjectID: access.ProjectID, Alias: access.Alias, DatabaseName: access.DatabaseName, ReadOnly: true}
		if access.Server != nil {
			view.ServerName = access.Server.Name
			view.ServerType = access.Server.Type
		}
		views = append(views, view)
	}
	return views, nil
}

func GetCodeDatabaseAccesses(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	projectID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || projectID == 0 {
		return c.JSON(e.Fail(errors.New("项目参数无效")))
	}
	if _, err := getCodeProjectWithPermission(uint(projectID), claims); err != nil {
		return c.JSON(e.Fail(err))
	}
	accesses, err := listCodeDatabaseAccesses(uint(projectID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(accesses))
}

func SaveCodeDatabaseAccess(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if claims.Role != constant.UserRoleAdmin && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("只有管理员可以配置数据库授权")))
	}
	projectID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || projectID == 0 {
		return c.JSON(e.Fail(errors.New("项目参数无效")))
	}
	if _, err := getCodeProjectWithPermission(uint(projectID), claims); err != nil {
		return c.JSON(e.Fail(err))
	}
	var req struct {
		ID           uint   `json:"id"`
		ServerID     uint   `json:"serverId"`
		DatabaseName string `json:"databaseName"`
		Alias        string `json:"alias"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	req.Alias = strings.TrimSpace(req.Alias)
	req.DatabaseName = strings.TrimSpace(req.DatabaseName)
	if req.ServerID == 0 || req.DatabaseName == "" || req.Alias == "" || len(req.Alias) > 64 || len(req.DatabaseName) > 255 {
		return c.JSON(e.Fail(errors.New("数据库授权参数无效")))
	}
	if _, err := repo.NewDatabaseServer().Get(req.ServerID); err != nil {
		return c.JSON(e.Fail(errors.New("数据库服务器不存在")))
	}
	access := model.AICodeDatabaseAccess{ID: req.ID, ProjectID: uint(projectID), ServerID: req.ServerID, DatabaseName: req.DatabaseName, Alias: req.Alias, ReadOnly: true}
	if access.ID == 0 {
		if err := global.DB.Create(&access).Error; err != nil {
			return c.JSON(e.Fail(err))
		}
	} else {
		result := global.DB.Model(&model.AICodeDatabaseAccess{}).
			Where("id = ? AND project_id = ?", access.ID, access.ProjectID).
			Updates(map[string]interface{}{
				"server_id": access.ServerID, "database_name": access.DatabaseName,
				"alias": access.Alias, "read_only": true,
			})
		if result.Error != nil {
			return c.JSON(e.Fail(result.Error))
		}
		if result.RowsAffected == 0 {
			return c.JSON(e.Fail(errors.New("数据库授权不存在")))
		}
	}
	accesses, err := listCodeDatabaseAccesses(uint(projectID))
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(accesses))
}

func DeleteCodeDatabaseAccess(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	if claims.Role != constant.UserRoleAdmin && claims.Role != constant.UserRoleSuper {
		return c.JSON(e.Fail(errors.New("只有管理员可以删除数据库授权")))
	}
	projectID, projectErr := strconv.ParseUint(c.Params("id"), 10, 64)
	accessID, accessErr := strconv.ParseUint(c.Params("accessId"), 10, 64)
	if projectErr != nil || accessErr != nil || projectID == 0 || accessID == 0 {
		return c.JSON(e.Fail(errors.New("数据库授权参数无效")))
	}
	if _, err := getCodeProjectWithPermission(uint(projectID), claims); err != nil {
		return c.JSON(e.Fail(err))
	}
	result := global.DB.Where("id = ? AND project_id = ?", accessID, projectID).Delete(&model.AICodeDatabaseAccess{})
	if result.Error != nil {
		return c.JSON(e.Fail(result.Error))
	}
	if result.RowsAffected == 0 {
		return c.JSON(e.Fail(errors.New("数据库授权不存在")))
	}
	return c.JSON(e.Succ(nil))
}

func ExecuteCodeDatabaseQuery(c fiber.Ctx) error {
	startedAt := time.Now()
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return c.JSON(e.Fail(errors.New("会话 ID 无效")))
	}
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var req struct {
		AccessID uint   `json:"accessId"`
		SQL      string `json:"sql"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	var access model.AICodeDatabaseAccess
	if err := global.DB.Where("id = ? AND project_id = ? AND read_only = ?", req.AccessID, session.ProjectID, true).First(&access).Error; err != nil {
		return c.JSON(e.Fail(errors.New("数据库授权不存在或不属于当前项目")))
	}
	fingerprint := codeDatabaseSQLFingerprint(req.SQL)
	if err := allowCodeDatabaseQuery(claims.UserId, session.ProjectID); err != nil {
		recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "database_query", "denied", access.Alias, err.Error(), c.IP(), startedAt, codeAuditMeta{"sqlFingerprint": fingerprint})
		return c.JSON(e.Fail(err))
	}
	result, err := service.NewDBManagerService().ExecCodeReadOnlySQL(access.ServerID, access.DatabaseName, req.SQL)
	if err != nil {
		recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "database_query", "failed", access.Alias, err.Error(), c.IP(), startedAt, codeAuditMeta{"sqlFingerprint": fingerprint})
		return c.JSON(e.Fail(err))
	}
	recordCodeAudit(claims.UserId, session.ProjectID, session.ID, "database_query", "success", access.Alias, "只读查询完成", c.IP(), startedAt, codeAuditMeta{"sqlFingerprint": fingerprint, "truncated": result["truncated"], "limit": result["limit"]})
	return c.JSON(e.Succ(result))
}

func codeDatabaseSQLFingerprint(statement string) string {
	normalized := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(statement))), " ")
	digest := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(digest[:])
}
