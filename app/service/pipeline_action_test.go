package service

import "testing"

func TestNormalizePipelineActionType(t *testing.T) {
	tests := map[string]string{
		"":            "none",
		"none":        "none",
		"deploy":      "none",
		"build":       "build_image",
		"build_image": "build_image",
	}
	for input, expected := range tests {
		if actual := normalizePipelineActionType(input); actual != expected {
			t.Fatalf("normalizePipelineActionType(%q) = %q, want %q", input, actual, expected)
		}
	}
}
