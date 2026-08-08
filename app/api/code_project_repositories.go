package api

import (
	"path/filepath"

	"github.com/aihop/gopanel/app/e"
	"github.com/aihop/gopanel/constant"
	"github.com/aihop/gopanel/utils/token"
	"github.com/gofiber/fiber/v3"
)

type codeProjectRepositoryOption struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func DiscoverCodeProjectRepositories(c fiber.Ctx) error {
	claims := c.Locals(constant.AppAuthName).(*token.CustomClaims)
	var req struct {
		SourceDirs []string `json:"sourceDirs"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return c.JSON(e.Fail(err))
	}
	sourceDirs, err := normalizeAIProjectSourceDirs(req.SourceDirs, claims)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	candidates, err := discoverCodeRepositoryCandidates(sourceDirs)
	if err != nil {
		return c.JSON(e.Fail(err))
	}
	return c.JSON(e.Succ(codeProjectRepositoryOptions(candidates)))
}

func codeProjectRepositoryOptions(candidates []codeRepositoryCandidate) []codeProjectRepositoryOption {
	options := make([]codeProjectRepositoryOption, 0, len(candidates))
	for _, candidate := range candidates {
		options = append(options, codeProjectRepositoryOption{
			Name: filepath.Base(candidate.SourceDir),
			Path: candidate.SourceDir,
		})
	}
	return options
}
