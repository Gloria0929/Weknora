package handler

import (
	"strings"
	"testing"
)

// The samples mirror what Git actually prints, including the transport trace
// lines a leftover GIT_CURL_VERBOSE would add. Authentication results are
// detected on the raw text, so a stripped trace line must not decide the
// outcome.
func TestGitPushAuthFailureDetection(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name: "rejected credential",
			output: "remote: Unauthorized\n" +
				"fatal: Authentication failed for 'https://gitee.com/team/test-work.git/'",
			want: true,
		},
		{
			name: "askpass noise hides the server result",
			output: "09:28:10.234484 http.c:623              => Send header: GET /team/test-work.git/info/refs?service=git-receive-pack HTTP/1.1\n" +
				"09:28:10.296287 http.c:623              <= Recv header: HTTP/1.1 401 Unauthorized\n" +
				"error: unable to read askpass response from '/bin/false'\n" +
				"fatal: could not read Username for 'https://gitee.com': terminal prompts disabled",
			want: true,
		},
		{
			name:   "prompt disabled after no credential was offered",
			output: "fatal: could not read Password for 'https://gitee.com': terminal prompts disabled",
			want: true,
		},
		{
			name:   "permission denied over ssh",
			output: "git@gitee.com: Permission denied (publickey).\nfatal: Could not read from remote repository.",
			want: true,
		},
		{
			name:   "status line without a credential error",
			output: "error: failed to push some refs to 'https://gitee.com/team/test-work.git'",
			want: false,
		},
		{
			name:   "non-fast-forward rejection with http 200 trace",
			output: "09:28:10.296287 http.c:623              <= Recv header: HTTP/1.1 200 OK\n" +
				" ! [rejected]        main -> main (non-fast-forward)",
			want: false,
		},
		{
			name:   "empty output",
			output: "",
			want:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := gitPushAuthFailed(tc.output); got != tc.want {
				t.Fatalf("gitPushAuthFailed() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGitPushFailureDetailDropsTransportTrace(t *testing.T) {
	raw := "09:28:10.234479 http.c:611              => Send header, 0000000249 bytes (0x000000f9)\n" +
		"09:28:10.296287 http.c:623              <= Recv header: HTTP/1.1 401 Unauthorized\n" +
		"remote: Unauthorized\n" +
		"fatal: Authentication failed for 'https://gitee.com/team/test-work.git/'"
	detail := gitPushFailureDetail(raw)
	if strings.Contains(detail, "http.c:") {
		t.Fatalf("trace lines must be stripped, got %q", detail)
	}
	if want := "remote: Unauthorized\nfatal: Authentication failed for 'https://gitee.com/team/test-work.git/'"; detail != want {
		t.Fatalf("detail = %q, want %q", detail, want)
	}
}
