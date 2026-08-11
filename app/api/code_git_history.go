package api

import (
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/aihop/gopanel/app/model"
)

type codeGitHistoryCommit struct {
	Commit      string    `json:"commit"`
	ShortCommit string    `json:"shortCommit"`
	Author      string    `json:"author"`
	AuthoredAt  time.Time `json:"authoredAt"`
	Subject     string    `json:"subject"`
	Merged      bool      `json:"merged"`
}

type codeGitHistoryRepository struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Branch       string                 `json:"branch"`
	TargetBranch string                 `json:"targetBranch"`
	BaseCommit   string                 `json:"baseCommit"`
	ResultCommit string                 `json:"resultCommit"`
	Commits      []codeGitHistoryCommit `json:"commits"`
}

type codeGitHistory struct {
	Available    bool                       `json:"available"`
	Repositories []codeGitHistoryRepository `json:"repositories"`
	Commits      int                        `json:"commits"`
}

func loadCodeGitHistory(session *model.AIDevSession, excludedRepositories []string) (codeGitHistory, error) {
	repositories := discoverCodeGitResultRepositories(session, excludedRepositories)
	result := codeGitHistory{
		Available:    len(repositories) > 0,
		Repositories: make([]codeGitHistoryRepository, 0, len(repositories)),
	}
	for _, repository := range repositories {
		output, truncated, err := runCodeGitReviewCommand(
			repository.root, false, codeGitDiffOutputLimit,
			"log", "--format=format:%H%x1f%h%x1f%an%x1f%aI%x1f%s%x1e",
			repository.BaseCommit+".."+repository.ResultCommit,
		)
		if err != nil {
			return codeGitHistory{}, err
		}
		if truncated {
			return codeGitHistory{}, errors.New("Git 提交历史过大，无法完整展示")
		}
		commits := parseCodeGitHistoryCommits(output)
		for index := range commits {
			commits[index].Merged = codeGitCommitMerged(repository, commits[index].Commit)
		}
		result.Commits += len(commits)
		result.Repositories = append(result.Repositories, codeGitHistoryRepository{
			ID: repository.ID, Name: repository.Name, Branch: repository.Branch,
			TargetBranch: repository.targetBranch,
			BaseCommit:   repository.BaseCommit, ResultCommit: repository.ResultCommit, Commits: commits,
		})
	}
	return result, nil
}

func codeGitCommitMerged(repository codeGitRepository, commit string) bool {
	if repository.targetBranch == "" || commit == "" {
		return false
	}
	_, err := runCodeGit(
		repository.root, "merge-base", "--is-ancestor", commit,
		"refs/heads/"+repository.targetBranch,
	)
	return err == nil
}

func parseCodeGitHistoryCommits(output string) []codeGitHistoryCommit {
	records := strings.Split(output, "\x1e")
	commits := make([]codeGitHistoryCommit, 0, len(records))
	for _, record := range records {
		fields := strings.Split(strings.TrimSpace(record), "\x1f")
		if len(fields) != 5 {
			continue
		}
		authoredAt, err := time.Parse(time.RFC3339, fields[3])
		if err != nil {
			continue
		}
		commits = append(commits, codeGitHistoryCommit{
			Commit: fields[0], ShortCommit: fields[1], Author: fields[2], AuthoredAt: authoredAt, Subject: fields[4],
		})
	}
	return commits
}

func loadCodeGitHistoryDiff(
	session *model.AIDevSession, excludedRepositories []string, repositoryID, commit string,
) (string, bool, error) {
	repository, err := findCodeGitRepository(
		discoverCodeGitResultRepositories(session, excludedRepositories), strings.TrimSpace(repositoryID),
	)
	if err != nil {
		return "", false, err
	}
	commit = strings.TrimSpace(commit)
	if !isCodeGitCommitHash(commit) || !codeGitCommitInResultRange(*repository, commit) {
		return "", false, errors.New("提交不在当前任务分支历史中")
	}
	return runCodeGitReviewCommand(
		repository.root, false, codeGitDiffOutputLimit,
		"show", "--first-parent", "--format=fuller", "--no-ext-diff", "--unified=3", commit,
	)
}

func isCodeGitCommitHash(commit string) bool {
	if len(commit) != 40 {
		return false
	}
	_, err := hex.DecodeString(commit)
	return err == nil
}

func codeGitCommitInResultRange(repository codeGitRepository, commit string) bool {
	if commit == repository.BaseCommit {
		return false
	}
	if _, err := runCodeGit(repository.root, "merge-base", "--is-ancestor", repository.BaseCommit, commit); err != nil {
		return false
	}
	_, err := runCodeGit(repository.root, "merge-base", "--is-ancestor", commit, repository.ResultCommit)
	return err == nil
}
