package api

import "testing"

func TestPrepareCodeRepositoryUsesLocalCommitWhenRemoteUnavailable(t *testing.T) {
	localDir, _ := createCodeRemoteRepository(t)
	localCommit, err := runCodeGit(localDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCodeGit(localDir, "remote", "set-url", "origin", "https://127.0.0.1:1/gopanel.git"); err != nil {
		t.Fatal(err)
	}

	prepared, err := prepareCodeRepository(localDir)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.BaseCommit != localCommit || prepared.RemoteName != "origin" || prepared.SyncStatus != "offline" {
		t.Fatalf("unexpected offline repository: %#v", prepared)
	}
}

func TestPrepareCodeRepositoryUsesLocalCommitWhenBoundCredentialUnavailable(t *testing.T) {
	withCodeGovernanceDB(t)
	localDir, _ := createCodeRemoteRepository(t)
	localCommit, err := runCodeGit(localDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := prepareCodeRepositoryCandidateForBranchWithCredential(
		codeRepositoryCandidate{SourceDir: localDir}, false, "", 999999,
	)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.BaseCommit != localCommit || prepared.RemoteName != "origin" || prepared.SyncStatus != "offline" {
		t.Fatalf("unexpected offline repository with unavailable credential: %#v", prepared)
	}
}
