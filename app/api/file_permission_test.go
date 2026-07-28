package api

import "testing"

func TestValidFileTaskKeyRejectsPathTraversal(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{key: "download_12_abcDEF123", want: true},
		{key: "download_12_../secret", want: false},
		{key: "download_12_%2e%2e", want: false},
		{key: "compress_12_abc", want: false},
		{key: "download_", want: false},
	}
	for _, tt := range tests {
		if got := validFileTaskKey(tt.key, "download_"); got != tt.want {
			t.Fatalf("validFileTaskKey(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}
