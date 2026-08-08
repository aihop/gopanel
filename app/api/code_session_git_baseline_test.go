package api

import "testing"

func TestCodeSessionGitSyncRequiresConfirmationForStaleBaselineAtRemoteHead(t *testing.T) {
	session, remoteDir := createCodeSessionSyncFixture(t)
	targets, err := codeSessionGitSyncTargets(session)
	if err != nil {
		t.Fatal(err)
	}
	baseCommit := targets[0].BaseCommit
	updater := cloneCodeRepository(t, remoteDir)
	remoteCommit := commitCodeTestFile(t, updater, "remote-baseline.txt", "remote baseline\n")
	if _, err := runCodeGit(updater, "push", "origin", "HEAD"); err != nil {
		t.Fatal(err)
	}
	fetchErrors := fetchCodeSessionGitTargets(targets)
	if fetchErrors[targets[0].ID] != nil {
		t.Fatal(fetchErrors[targets[0].ID])
	}
	if _, err := runCodeGit(targets[0].WorktreeDir, "merge", "--ff-only", codeSessionGitRemoteRef(targets[0])); err != nil {
		t.Fatal(err)
	}
	if baseCommit == remoteCommit {
		t.Fatal("fixture did not advance the remote baseline")
	}

	status := inspectCodeSessionGitSyncTargets(session.ID, targets, fetchErrors)
	repository := status.Repositories[0]
	if status.Status != "integrated" || repository.Status != "integrated" || !repository.CanSync {
		t.Fatalf("stale baseline at remote HEAD should require confirmation: %#v", status)
	}
	if repository.LocalCommit != remoteCommit || repository.RemoteCommit != remoteCommit || repository.Reason != "remote_integrated" {
		t.Fatalf("unexpected integrated baseline state: %#v", repository)
	}

	targets[0].BaseCommit = remoteCommit
	confirmed := inspectCodeSessionGitSyncTargets(session.ID, targets, fetchErrors)
	if confirmed.Status != "synced" || confirmed.Repositories[0].Status != "synced" || confirmed.CanSync {
		t.Fatalf("confirmed baseline should be synced: %#v", confirmed)
	}
}
