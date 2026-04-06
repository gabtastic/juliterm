package tmux

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// ===========================================================================
// ParseGitRemoteURL
// ===========================================================================

func TestParseGitRemoteURL_SSH(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"git@github.com:cihann/juliterm.git", "cihann/juliterm"},
		{"git@github.com:cihann/juliterm", "cihann/juliterm"},
		{"git@gitlab.com:org/project.git", "org/project"},
	}
	for _, tt := range tests {
		got := ParseGitRemoteURL(tt.input)
		if got != tt.want {
			t.Errorf("ParseGitRemoteURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseGitRemoteURL_HTTPS(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://github.com/cihann/juliterm.git", "cihann/juliterm"},
		{"https://github.com/cihann/juliterm", "cihann/juliterm"},
		{"http://gitlab.com/org/project.git", "org/project"},
	}
	for _, tt := range tests {
		got := ParseGitRemoteURL(tt.input)
		if got != tt.want {
			t.Errorf("ParseGitRemoteURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseGitRemoteURL_Invalid(t *testing.T) {
	tests := []string{
		"",
		"not-a-url",
		"/local/path",
	}
	for _, input := range tests {
		got := ParseGitRemoteURL(input)
		if got != "" {
			t.Errorf("ParseGitRemoteURL(%q) = %q, want empty", input, got)
		}
	}
}

// ===========================================================================
// DeriveSessionStatus
// ===========================================================================

func TestDeriveSessionStatus(t *testing.T) {
	tests := []struct {
		attached        int
		activePaneDead  bool
		hasChildProcess bool
		want            string
	}{
		{0, false, false, "detached"},
		{0, true, true, "detached"}, // detached takes priority
		{1, true, false, "error"},   // dead pane
		{1, false, false, "idle"},   // no child process
		{1, false, true, "running"}, // attached, alive, has child
		{2, false, true, "running"}, // multiple clients
	}
	for _, tt := range tests {
		got := DeriveSessionStatus(tt.attached, tt.activePaneDead, tt.hasChildProcess)
		if got != tt.want {
			t.Errorf("DeriveSessionStatus(%d, %v, %v) = %q, want %q",
				tt.attached, tt.activePaneDead, tt.hasChildProcess, got, tt.want)
		}
	}
}

// ===========================================================================
// DetectToolType (unit-testable parts)
// ===========================================================================

// Note: DetectToolType relies on `ps` and cannot be fully unit-tested without
// a running process tree. We test the pattern-matching logic indirectly via
// the exported DeriveSessionStatus and ParseGitRemoteURL, and integration-test
// DetectToolType manually.

// ===========================================================================
// UpdateSessionMetadata
// ===========================================================================

func TestSessionTree_UpdateSessionMetadata(t *testing.T) {
	tree := NewSessionTree()
	tree.UpdateSession(makeTestSession("$0", "main", "%0"))

	tree.UpdateSessionMetadata("$0", func(s *Session) {
		s.Status = "running"
		s.ToolType = "claude"
		s.GitBranch = "feat/sidebar"
		s.GitRepo = "cihann/juliterm"
		s.WorkingDir = "/home/user/project"
		s.LastActivity = 1711111111
	})

	got := tree.GetSession("$0")
	if got.Status != "running" {
		t.Errorf("Status = %q, want running", got.Status)
	}
	if got.ToolType != "claude" {
		t.Errorf("ToolType = %q, want claude", got.ToolType)
	}
	if got.GitBranch != "feat/sidebar" {
		t.Errorf("GitBranch = %q, want feat/sidebar", got.GitBranch)
	}
	if got.GitRepo != "cihann/juliterm" {
		t.Errorf("GitRepo = %q, want cihann/juliterm", got.GitRepo)
	}
	if got.WorkingDir != "/home/user/project" {
		t.Errorf("WorkingDir = %q, want /home/user/project", got.WorkingDir)
	}
	if got.LastActivity != 1711111111 {
		t.Errorf("LastActivity = %d, want 1711111111", got.LastActivity)
	}
	// Windows should be preserved.
	if len(got.Windows) != 1 {
		t.Errorf("Windows count = %d, want 1", len(got.Windows))
	}
}

func TestSessionTree_UpdateSessionMetadata_MonitoringFields(t *testing.T) {
	tree := NewSessionTree()
	tree.UpdateSession(makeTestSession("$0", "main", "%0"))

	tree.UpdateSessionMetadata("$0", func(s *Session) {
		s.ActivityState = ActivityExecuting
		s.ActivitySummary = "Running tests in juli"
	})

	got := tree.GetSession("$0")
	if got.ActivityState != ActivityExecuting {
		t.Errorf("ActivityState = %q, want %q", got.ActivityState, ActivityExecuting)
	}
	if got.ActivitySummary != "Running tests in juli" {
		t.Errorf("ActivitySummary = %q, want %q", got.ActivitySummary, "Running tests in juli")
	}
	if len(got.Windows) != 1 {
		t.Errorf("Windows count = %d, want 1", len(got.Windows))
	}
}

func TestSessionTree_UpdateSessionMetadata_NonExistent(t *testing.T) {
	tree := NewSessionTree()
	// Should not panic.
	tree.UpdateSessionMetadata("$99", func(s *Session) {
		s.Status = "running"
	})
}

// ===========================================================================
// Parse with metadata fields
// ===========================================================================

func TestParseSessionList_WithMetadata(t *testing.T) {
	output := "$0\tmain\t1711111111\t2\n$1\tdev\t1711111222\t0\n"
	sessions, err := ParseSessionList(output)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].LastActivity != 1711111111 {
		t.Errorf("session[0].LastActivity = %d, want 1711111111", sessions[0].LastActivity)
	}
	if sessions[0].Attached != 2 {
		t.Errorf("session[0].Attached = %d, want 2", sessions[0].Attached)
	}
	if sessions[1].Attached != 0 {
		t.Errorf("session[1].Attached = %d, want 0", sessions[1].Attached)
	}
}

func TestParsePaneList_WithMetadata(t *testing.T) {
	output := "%0\t80\t24\t1\tclaude\t/home/user/project\t54321\n"
	panes, err := ParsePaneList(output)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(panes) != 1 {
		t.Fatalf("expected 1 pane, got %d", len(panes))
	}
	if panes[0].CurrentCommand != "claude" {
		t.Errorf("CurrentCommand = %q, want claude", panes[0].CurrentCommand)
	}
	if panes[0].CurrentPath != "/home/user/project" {
		t.Errorf("CurrentPath = %q, want /home/user/project", panes[0].CurrentPath)
	}
	if panes[0].Pid != 54321 {
		t.Errorf("Pid = %d, want 54321", panes[0].Pid)
	}
}

func TestCollectActivePaneInfo_PrefersActivePaneInActiveWindow(t *testing.T) {
	output := "" +
		"$0\t%old\t0\t1\t111\t/tmp/old\tgo\t0\n" +
		"$0\t%inactive\t1\t0\t222\t/tmp/inactive\tbash\t0\n" +
		"$0\t%current\t1\t1\t333\t/tmp/current\tgo\t0\n"

	activePanes := collectActivePaneInfo(output)
	info := activePanes["$0"]
	if info == nil {
		t.Fatalf("expected active pane info for session")
	}
	if info.paneID != "%current" {
		t.Fatalf("paneID = %q, want %%current", info.paneID)
	}
	if info.pid != 333 {
		t.Fatalf("pid = %d, want 333", info.pid)
	}
	if info.path != "/tmp/current" {
		t.Fatalf("path = %q, want /tmp/current", info.path)
	}
	if info.command != "go" {
		t.Fatalf("command = %q, want go", info.command)
	}
}

func TestBuildMonitoringInputs_UsesProcessArgsFromSnapshot(t *testing.T) {
	session := &Session{
		ID:           "$0",
		Name:         "main",
		Attached:     1,
		ToolType:     "unknown",
		GitRepo:      "cihann/juliterm",
		GitBranch:    "main",
		WorkingDir:   "/home/user/juli/internal/server",
		LastActivity: 1711111111,
	}
	info := &activePaneInfo{
		paneID:  "%0",
		pid:     4321,
		path:    session.WorkingDir,
		command: "go",
	}
	snapshot := ProcessSnapshot{
		4321: {Comm: "go", Args: "go test ./internal/server"},
	}

	input := buildMonitoringInputs(session, info, snapshot, "=== RUN   TestLogin\n", time.Unix(1711111222, 0))
	if input.CommandArgs != "go test ./internal/server" {
		t.Fatalf("CommandArgs = %q, want %q", input.CommandArgs, "go test ./internal/server")
	}

	got := DeriveMonitoringResult(input)
	want := MonitoringResult{
		State:   ActivityExecuting,
		Summary: "Running Go tests in internal/server",
	}
	if got != want {
		t.Fatalf("DeriveMonitoringResult() = %#v, want %#v", got, want)
	}
}

func TestCaptureMonitoringExcerptWithRunner_UsesShortTimeoutAndSmallBounds(t *testing.T) {
	var gotArgs []string
	excerpt := captureMonitoringExcerptWithRunner(context.Background(), "tmux", "%9", func(ctx context.Context, tmuxPath string, args ...string) ([]byte, error) {
		if tmuxPath != "tmux" {
			t.Fatalf("tmuxPath = %q, want tmux", tmuxPath)
		}
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("expected per-call deadline")
		}
		if remaining := time.Until(deadline); remaining <= 0 || remaining > monitoringExcerptTimeout {
			t.Fatalf("timeout remaining = %v, want 0 < remaining <= %v", remaining, monitoringExcerptTimeout)
		}
		gotArgs = append([]string(nil), args...)
		return []byte("top\nbottom\n\n"), nil
	})

	if excerpt != "top\nbottom" {
		t.Fatalf("excerpt = %q, want %q", excerpt, "top\nbottom")
	}

	wantArgs := []string{"capture-pane", "-p", "-t", "%9", "-S", "-6"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("args len = %d, want %d (%v)", len(gotArgs), len(wantArgs), gotArgs)
	}
	for i := range wantArgs {
		if gotArgs[i] != wantArgs[i] {
			t.Fatalf("args[%d] = %q, want %q", i, gotArgs[i], wantArgs[i])
		}
	}
}

func TestCaptureMonitoringExcerptWithRunner_SoftFails(t *testing.T) {
	excerpt := captureMonitoringExcerptWithRunner(context.Background(), "tmux", "%9", func(ctx context.Context, tmuxPath string, args ...string) ([]byte, error) {
		return nil, errors.New("boom")
	})
	if excerpt != "" {
		t.Fatalf("excerpt = %q, want empty string", excerpt)
	}
}

func TestSessionTree_ToJSON_IncludesMonitoringFields(t *testing.T) {
	tree := NewSessionTree()
	tree.UpdateSession(makeTestSession("$0", "alpha", "%0"))
	tree.UpdateSessionMetadata("$0", func(s *Session) {
		s.ActivityState = ActivityWaiting
		s.ActivitySummary = "Waiting for confirmation in codex"
	})

	data, err := tree.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	var sessions []map[string]any
	if err := json.Unmarshal(data, &sessions); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if got := sessions[0]["activityState"]; got != string(ActivityWaiting) {
		t.Errorf("activityState = %#v, want %q", got, ActivityWaiting)
	}
	if got := sessions[0]["activitySummary"]; got != "Waiting for confirmation in codex" {
		t.Errorf("activitySummary = %#v, want %q", got, "Waiting for confirmation in codex")
	}
}
