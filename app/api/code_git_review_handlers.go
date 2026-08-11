package api

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/app/model"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

func getCodeGitSessionContext(c fiber.Ctx) (*model.AIDevSession, *model.AIProject, error) {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return nil, nil, errors.New("会话 ID 无效")
	}
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return nil, nil, err
	}
	if err := validateAIProjectWorkDirForClaims(session.WorkDir, claims); err != nil {
		return nil, nil, err
	}
	if session.ProjectID == 0 {
		return session, &model.AIProject{}, nil
	}
	project, err := getCodeProjectWithPermission(session.ProjectID, claims)
	return session, project, err
}

func getCodeGitResultSessionContext(c fiber.Ctx) (*model.AIDevSession, *model.AIProject, error) {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	sessionID, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil || sessionID == 0 {
		return nil, nil, errors.New("会话 ID 无效")
	}
	session, err := getAISessionWithPermission(uint(sessionID), claims)
	if err != nil {
		return nil, nil, err
	}
	if session.ProjectID == 0 {
		if err := validateAIProjectWorkDirForClaims(session.WorkDir, claims); err != nil {
			return nil, nil, err
		}
		return session, &model.AIProject{}, nil
	}
	project, err := getCodeProjectWithPermission(session.ProjectID, claims)
	if err != nil {
		return nil, nil, err
	}
	for _, sourceDir := range project.SourceDirs {
		if isCodeRepositoryExcluded(sourceDir, project.ExcludedRepositories) {
			continue
		}
		if err := validateAIProjectWorkDirForClaims(sourceDir, claims); err != nil {
			return nil, nil, err
		}
	}
	return session, project, nil
}

func GetCodeGitStatus(c fiber.Ctx) error {
	scope := c.Query("scope", "workspace")
	var session *model.AIDevSession
	var project *model.AIProject
	var err error
	if scope == "result" {
		session, project, err = getCodeGitResultSessionContext(c)
	} else {
		session, project, err = getCodeGitSessionContext(c)
	}
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var status codeGitStatus
	if scope == "result" {
		status, err = loadCodeGitResultStatus(session, project.ExcludedRepositories)
	} else {
		status, err = loadCodeGitStatus(session, project.SourceDirs, project.ExcludedRepositories)
		status.Scope = "workspace"
	}
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(status))
}

func GetCodeGitDiff(c fiber.Ctx) error {
	scope := c.Query("scope", "workspace")
	var session *model.AIDevSession
	var project *model.AIProject
	var err error
	if scope == "result" {
		session, project, err = getCodeGitResultSessionContext(c)
	} else {
		session, project, err = getCodeGitSessionContext(c)
	}
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	if scope == "result" {
		content, truncated, resultErr := loadCodeGitResultFileDiff(
			session, project.ExcludedRepositories, c.Query("repositoryId"), c.Query("path"),
		)
		if resultErr != nil {
			return c.JSON(e.Fail(resultErr))
		}
		return c.JSON(e.Succ(fiber.Map{
			"repositoryId": c.Query("repositoryId"), "path": c.Query("path"), "kind": "result",
			"scope": scope, "content": content, "truncated": truncated,
		}))
	}
	repository, err := findCodeGitRepository(
		discoverCodeGitRepositories(session, project.SourceDirs, project.ExcludedRepositories), c.Query("repositoryId"),
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	kind := c.Query("kind", "working")
	if kind != "working" && kind != "staged" {
		return c.JSON(e.Fail(errors.New("Git 差异类型无效")))
	}
	file, err := findCodeGitFile(*repository, c.Query("path"), kind)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	content, truncated, err := loadCodeGitDiff(*repository, *file, kind)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{"repositoryId": repository.ID, "path": file.Path, "kind": kind, "content": content, "truncated": truncated}))
}

func GetCodeGitHistory(c fiber.Ctx) error {
	session, project, err := getCodeGitResultSessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	history, err := loadCodeGitHistory(session, project.ExcludedRepositories)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(history))
}

func GetCodeGitHistoryDiff(c fiber.Ctx) error {
	session, project, err := getCodeGitResultSessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	content, truncated, err := loadCodeGitHistoryDiff(
		session, project.ExcludedRepositories, c.Query("repositoryId"), c.Query("commit"),
	)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(fiber.Map{
		"repositoryId": c.Query("repositoryId"), "commit": c.Query("commit"),
		"content": content, "truncated": truncated,
	}))
}

func UpdateCodeGitStage(c fiber.Ctx) error {
	var req struct {
		RepositoryID string   `json:"repositoryId"`
		Paths        []string `json:"paths"`
		Staged       bool     `json:"staged"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	if len(req.Paths) == 0 || len(req.Paths) > 200 {
		return c.JSON(e.Fail(errors.New("请选择 1 到 200 个 Git 文件")))
	}
	session, project, err := getCodeGitSessionContext(c)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	var updated codeGitStatus
	err = runCodeSessionGitMutation(session, func(current *model.AIDevSession) error {
		repository, mutationErr := findCodeGitRepository(
			discoverCodeGitRepositories(current, project.SourceDirs, project.ExcludedRepositories),
			req.RepositoryID,
		)
		if mutationErr != nil {
			return mutationErr
		}
		status, mutationErr := loadCodeGitRepositoryStatus(*repository)
		if mutationErr != nil {
			return mutationErr
		}
		validPaths := make(map[string]codeGitFile, len(status.Files))
		for _, file := range status.Files {
			validPaths[filepath.ToSlash(file.Path)] = file
		}
		paths := make([]string, 0, len(req.Paths))
		seen := make(map[string]struct{})
		for _, requestedPath := range req.Paths {
			cleanPath := filepath.ToSlash(path.Clean(strings.TrimSpace(requestedPath)))
			file, exists := validPaths[cleanPath]
			if !exists || (req.Staged && !file.Changed && !file.Untracked) || (!req.Staged && !file.Staged) {
				return fmt.Errorf("文件不允许执行当前暂存操作：%s", cleanPath)
			}
			if _, exists := seen[cleanPath]; exists {
				continue
			}
			seen[cleanPath] = struct{}{}
			paths = append(paths, cleanPath)
		}
		if mutationErr = updateCodeGitPathsStage(*repository, paths, req.Staged); mutationErr != nil {
			return mutationErr
		}
		updated, mutationErr = loadCodeGitStatus(current, project.SourceDirs, project.ExcludedRepositories)
		return mutationErr
	})
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(updated))
}
