package api

import "testing"

func TestBuildAIAgentExecArgsDoesNotUseShell(t *testing.T) {
	payload := `"; touch /workspace/PWNED; echo "`
	args, err := buildAIAgentExecArgs("workspace", "trae", payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 6 {
		t.Fatalf("unexpected args: %#v", args)
	}
	if args[3] != "trae" || args[4] != "--message" || args[5] != payload {
		t.Fatalf("input was not preserved as one argument: %#v", args)
	}
	for _, arg := range args {
		if arg == "sh" || arg == "-c" {
			t.Fatalf("shell execution is not allowed: %#v", args)
		}
	}
}

func TestNormalizeAIAgentNameRejectsCommands(t *testing.T) {
	if _, err := normalizeAIAgentName("trae; id"); err == nil {
		t.Fatal("expected command-like agent name to be rejected")
	}
}
