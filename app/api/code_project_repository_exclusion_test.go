package api

import (
	"path/filepath"
	"testing"

	"github.com/aihop/gopanel/app/model"
)

func TestIsCodeRepositoryExcludedCoversNestedRepositories(t *testing.T) {
	excluded := normalizeCodeExcludedRepositories([]string{"/code/apay/app/themes/", "", "/code/apay/app/themes"})
	if len(excluded) != 1 {
		t.Fatalf("归一化后应去重成 1 项，实际 %#v", excluded)
	}
	if !isCodeRepositoryExcluded("/code/apay/app/themes", excluded) {
		t.Fatal("排除项自身应命中")
	}
	// 排除一个目录自然排除它内部的仓库，不用逐个列。
	if !isCodeRepositoryExcluded("/code/apay/app/themes/nft", excluded) {
		t.Fatal("排除项内部的仓库应命中")
	}
	// 外层仓库不受影响：用户要的正是「保留 apay，去掉它的子仓库」。
	if isCodeRepositoryExcluded("/code/apay", excluded) {
		t.Fatal("父仓库不该被连坐")
	}
	if isCodeRepositoryExcluded("/code/qingpu-ai", excluded) {
		t.Fatal("无关仓库不该被排除")
	}
	// 前缀相同但不是同一棵树，不能误伤。
	if isCodeRepositoryExcluded("/code/apay/app/themes-backup", excluded) {
		t.Fatal("同前缀的兄弟目录不该被排除")
	}
}

func TestFilterExcludedCodeRepositoriesKeepsParentRepository(t *testing.T) {
	candidates := []codeRepositoryCandidate{
		{SourceDir: "/code/apay"},
		{SourceDir: "/code/apay/app/themes/nft", ParentSourceDir: "/code/apay", GitlinkPath: "app/themes/nft"},
		{SourceDir: "/code/apay/app/themes/panel", ParentSourceDir: "/code/apay", GitlinkPath: "app/themes/panel"},
		{SourceDir: "/code/qingpu-ai"},
	}
	kept := filterExcludedCodeRepositories(candidates, normalizeCodeExcludedRepositories([]string{"/code/apay/app/themes"}))
	if len(kept) != 2 {
		t.Fatalf("应只剩 apay 和 qingpu-ai，实际 %#v", kept)
	}
	for _, candidate := range kept {
		if filepath.Base(candidate.SourceDir) == "nft" || filepath.Base(candidate.SourceDir) == "panel" {
			t.Fatalf("被排除的子仓库仍在结果里: %s", candidate.SourceDir)
		}
	}
}

func TestFilterExcludedCodeRepositoriesIsNoopWithoutExclusions(t *testing.T) {
	candidates := []codeRepositoryCandidate{{SourceDir: "/code/apay"}}
	if got := filterExcludedCodeRepositories(candidates, nil); len(got) != 1 {
		t.Fatalf("没有排除项时不该过滤掉任何仓库: %#v", got)
	}
}

func TestValidateCodeExcludedRepositoriesProtectsPrimaryRepository(t *testing.T) {
	// 主交付仓库被排除会让交付无处落地，必须挡在保存这一步。
	err := validateCodeExcludedRepositories("/code/apay", normalizeCodeExcludedRepositories([]string{"/code/apay"}))
	if err == nil {
		t.Fatal("排除主交付仓库应当报错")
	}
	// 主交付仓库在被排除目录之内，同样要拦。
	err = validateCodeExcludedRepositories("/code/apay/app/themes/nft", normalizeCodeExcludedRepositories([]string{"/code/apay/app/themes"}))
	if err == nil {
		t.Fatal("主交付仓库位于排除目录内也应当报错")
	}
	if err := validateCodeExcludedRepositories("/code/apay", normalizeCodeExcludedRepositories([]string{"/code/apay/app/themes"})); err != nil {
		t.Fatalf("排除子仓库不该影响主交付仓库: %v", err)
	}
	if err := validateCodeExcludedRepositories("", []string{"/code/apay"}); err != nil {
		t.Fatalf("未设置主交付仓库时不该报错: %v", err)
	}
}

func TestDiscoverCodeProjectRepositoryCandidatesWithoutProjectKeepsEverything(t *testing.T) {
	// project 为空表示「配置态」：项目编辑页要列出全部仓库供用户勾选。
	var project *model.AIProject
	if _, err := discoverCodeProjectRepositoryCandidates(project, nil); err != nil {
		t.Fatalf("空目录不该报错: %v", err)
	}
}
