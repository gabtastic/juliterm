package tmux

import (
	"strings"
	"testing"
)

func TestDeriveActivityState_Priority(t *testing.T) {
	tests := []struct {
		name string
		sig  MonitoringSignals
		want ActivityState
	}{
		{
			name: "failure beats detached",
			sig:  MonitoringSignals{Attached: 0, LooksLikeFailure: true},
			want: ActivityNeedsAttention,
		},
		{
			name: "executing beats detached",
			sig:  MonitoringSignals{Attached: 0, HasRunningCommand: true},
			want: ActivityExecuting,
		},
		{
			name: "coding beats detached",
			sig:  MonitoringSignals{Attached: 0, HasAgentTool: true},
			want: ActivityCoding,
		},
		{
			name: "waiting beats detached",
			sig:  MonitoringSignals{Attached: 0, LooksWaitingForInput: true},
			want: ActivityWaiting,
		},
		{
			name: "needs attention beats executing",
			sig:  MonitoringSignals{Attached: 1, LooksLikeFailure: true, HasRunningCommand: true},
			want: ActivityNeedsAttention,
		},
		{
			name: "executing beats coding",
			sig:  MonitoringSignals{Attached: 1, HasRunningCommand: true, HasAgentTool: true},
			want: ActivityExecuting,
		},
		{
			name: "coding beats waiting",
			sig:  MonitoringSignals{Attached: 1, HasAgentTool: true, LooksWaitingForInput: true},
			want: ActivityCoding,
		},
		{
			name: "stable excerpt agent becomes waiting",
			sig:  MonitoringSignals{Attached: 1, HasAgentTool: true, LooksExcerptStable: true},
			want: ActivityWaiting,
		},
		{
			name: "stable excerpt without agent stays idle",
			sig:  MonitoringSignals{Attached: 1, LooksExcerptStable: true, LooksLikePromptIdle: true},
			want: ActivityIdle,
		},
		{
			name: "waiting beats idle",
			sig:  MonitoringSignals{Attached: 1, LooksWaitingForInput: true},
			want: ActivityWaiting,
		},
		{
			name: "idle beats unknown",
			sig:  MonitoringSignals{Attached: 1, LooksLikePromptIdle: true},
			want: ActivityIdle,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeriveActivityState(tt.sig); got != tt.want {
				t.Fatalf("DeriveActivityState() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractMonitoringSignals(t *testing.T) {
	input := MonitoringInputs{
		Attached:    1,
		ToolType:    "codex",
		Command:     "go",
		CommandArgs: "go test ./internal/server",
		WorkingDir:  "/home/user/juli/internal/server",
		GitRepo:     "cihann/juliterm",
		Excerpt:     "=== RUN   TestLogin\n--- FAIL: TestLogin (0.00s)\nFAIL\n",
	}

	got := ExtractMonitoringSignals(input)
	if !got.HasAgentTool {
		t.Fatalf("expected HasAgentTool to be true")
	}
	if !got.LooksLikeTestRun {
		t.Fatalf("expected LooksLikeTestRun to be true")
	}
	if !got.LooksLikeFailure {
		t.Fatalf("expected LooksLikeFailure to be true")
	}
	if got.ContextLabel != "internal/server" {
		t.Fatalf("ContextLabel = %q, want internal/server", got.ContextLabel)
	}
	if got.CommandLabel != "Go tests" {
		t.Fatalf("CommandLabel = %q, want Go tests", got.CommandLabel)
	}
}

func TestDeriveMonitoringResult_RealInputs(t *testing.T) {
	tests := []struct {
		name string
		in   MonitoringInputs
		want MonitoringResult
	}{
		{
			name: "running go tests in internal/server",
			in: MonitoringInputs{
				Attached:    1,
				Command:     "go",
				CommandArgs: "go test ./internal/server",
				WorkingDir:  "/home/user/juli/internal/server",
				Excerpt:     "=== RUN   TestLogin\n--- PASS: TestLogin (0.00s)\nok  github.com/cihann/juliterm/internal/server 0.010s\n",
			},
			want: MonitoringResult{
				State:   ActivityExecuting,
				Summary: "Running Go tests in internal/server",
			},
		},
		{
			name: "waiting for confirmation in codex",
			in: MonitoringInputs{
				Attached:    1,
				ToolType:    "codex",
				Command:     "codex",
				CommandArgs: "",
				WorkingDir:  "/home/user/juli",
				Excerpt:     "Press enter to confirm the next step",
			},
			want: MonitoringResult{
				State:   ActivityWaiting,
				Summary: "Waiting for confirmation in codex",
			},
		},
		{
			name: "test failures stay test failures",
			in: MonitoringInputs{
				Attached:    1,
				Command:     "go",
				CommandArgs: "go test ./internal/server",
				WorkingDir:  "/home/user/juli/internal/server",
				Excerpt:     "=== RUN   TestLogin\n--- FAIL: TestLogin (0.00s)\nFAIL\n",
			},
			want: MonitoringResult{
				State:   ActivityNeedsAttention,
				Summary: "Test failed in internal/server",
			},
		},
		{
			name: "build failures report the web app context",
			in: MonitoringInputs{
				Attached:    1,
				Command:     "npm",
				CommandArgs: "npm run build",
				WorkingDir:  "/home/user/juli/web",
				Excerpt:     "failed to compile",
			},
			want: MonitoringResult{
				State:   ActivityNeedsAttention,
				Summary: "Build failed in web app",
			},
		},
		{
			name: "completed test run with shell prompt does not stay executing",
			in: MonitoringInputs{
				Attached:   1,
				Command:    "zsh",
				WorkingDir: "/home/user/juli/internal/server",
				Excerpt:    "=== RUN   TestLogin\n--- PASS: TestLogin (0.00s)\nok  github.com/cihann/juliterm/internal/server 0.010s\nzsh$",
			},
			want: MonitoringResult{
				State:   ActivityIdle,
				Summary: "Idle at shell prompt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveMonitoringResult(tt.in)
			if got != tt.want {
				t.Fatalf("DeriveMonitoringResult() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDeriveMonitoringResult_StaleWaitingPromptDoesNotWinOverNewerOutput(t *testing.T) {
	input := MonitoringInputs{
		Attached:   1,
		ToolType:   "codex",
		Command:    "codex",
		WorkingDir: "/home/user/juli",
		Excerpt:    "Press enter to confirm the next step\nUpdated monitoring heuristics\n",
	}

	signals := ExtractMonitoringSignals(input)
	if signals.LooksWaitingForInput {
		t.Fatalf("expected stale waiting prompt to be ignored once newer output appears")
	}

	got := DeriveMonitoringResult(input)
	want := MonitoringResult{
		State:   ActivityCoding,
		Summary: "Editing files in juli",
	}
	if got != want {
		t.Fatalf("DeriveMonitoringResult() = %#v, want %#v", got, want)
	}
}

func TestDeriveMonitoringResult_StaleFailureDoesNotWinOverNewerPrompt(t *testing.T) {
	input := MonitoringInputs{
		Attached:   1,
		Command:    "zsh",
		WorkingDir: "/home/user/juli/internal/server",
		Excerpt:    "--- FAIL: TestLogin (0.00s)\nFAIL\nok  github.com/cihann/juliterm/internal/server 0.010s\nzsh$",
	}

	signals := ExtractMonitoringSignals(input)
	if signals.LooksLikeFailure {
		t.Fatalf("expected stale failure lines to be ignored once newer prompt output appears")
	}

	got := DeriveMonitoringResult(input)
	want := MonitoringResult{
		State:   ActivityIdle,
		Summary: "Idle at shell prompt",
	}
	if got != want {
		t.Fatalf("DeriveMonitoringResult() = %#v, want %#v", got, want)
	}
}

func TestExtractMonitoringSignals_InteractivePromptSuppressesGenericWork(t *testing.T) {
	got := ExtractMonitoringSignals(MonitoringInputs{
		Attached:   1,
		ToolType:   "codex",
		Command:    "codex",
		WorkingDir: "/home/user/juli",
		Excerpt:    "Press enter to confirm the next step",
	})

	if got.HasRunningCommand {
		t.Fatalf("expected HasRunningCommand to be false for an interactive prompt")
	}
	if got.HasAgentTool {
		t.Fatalf("expected HasAgentTool to be false while waiting")
	}
	if !got.LooksWaitingForInput {
		t.Fatalf("expected LooksWaitingForInput to be true")
	}
}

func TestDeriveMonitoringResult_VersionProcessTitleStaysCoding(t *testing.T) {
	// Claude Code sets process title to version (e.g. "2.1.87"),
	// which tmux reports as pane_current_command. This should NOT
	// be treated as a running command.
	input := MonitoringInputs{
		Attached:   1,
		ToolType:   "claude",
		Command:    "2.1.87",
		WorkingDir: "/home/user/juli",
	}

	signals := ExtractMonitoringSignals(input)
	if signals.HasRunningCommand {
		t.Fatalf("expected HasRunningCommand to be false for version process title")
	}
	if !signals.HasAgentTool {
		t.Fatalf("expected HasAgentTool to be true for agent with version process title")
	}

	got := DeriveMonitoringResult(input)
	if got.State != ActivityCoding {
		t.Fatalf("State = %q, want %q", got.State, ActivityCoding)
	}
}

func TestLooksLikeVersionString(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"2.1.87", true},
		{"1.0.0", true},
		{"22.14.0", true},
		{"0.1", true},
		{"node", false},
		{"claude", false},
		{"go", false},
		{"zsh", false},
		{"", false},
		{"123", false},      // no dot
		{".1.2", false},     // starts with dot
		{"abc.def", false},  // letters
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := looksLikeVersionString(tt.input); got != tt.want {
				t.Fatalf("looksLikeVersionString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestDeriveMonitoringResult_AgentLauncherCommandStaysCoding(t *testing.T) {
	input := MonitoringInputs{
		Attached:    1,
		ToolType:    "codex",
		Command:     "node",
		CommandArgs: "node /usr/local/bin/codex --approval-mode auto",
		WorkingDir:  "/home/user/juli",
	}

	signals := ExtractMonitoringSignals(input)
	if signals.HasRunningCommand {
		t.Fatalf("expected HasRunningCommand to be false for a codex launcher command")
	}
	if !signals.HasAgentTool {
		t.Fatalf("expected HasAgentTool to remain true for a codex launcher command")
	}

	got := DeriveMonitoringResult(input)
	want := MonitoringResult{
		State:   ActivityCoding,
		Summary: "Editing files in juli",
	}
	if got != want {
		t.Fatalf("DeriveMonitoringResult() = %#v, want %#v", got, want)
	}
}

func TestExtractMonitoringSignals_BenignCountsDoNotTriggerFailure(t *testing.T) {
	got := ExtractMonitoringSignals(MonitoringInputs{
		Attached:    1,
		Command:     "go",
		CommandArgs: "test ./internal/server",
		WorkingDir:  "/home/user/juli/internal/server",
		Excerpt:     "ok\n0 errors\nerrors: 0\n0 failures\n",
	})

	if got.LooksLikeFailure {
		t.Fatalf("expected LooksLikeFailure to be false for benign counts")
	}
	if got.State != "" {
		t.Fatalf("expected ExtractMonitoringSignals to leave State unset, got %q", got.State)
	}
}

func TestExtractMonitoringSignals_BenignConflictDoesNotTriggerFailure(t *testing.T) {
	got := ExtractMonitoringSignals(MonitoringInputs{
		Attached:    1,
		Command:     "go",
		CommandArgs: "test ./internal/server",
		WorkingDir:  "/home/user/juli/internal/server",
		Excerpt:     "0 conflicts\nconflicts: none\n",
	})

	if got.LooksLikeFailure {
		t.Fatalf("expected LooksLikeFailure to be false for benign conflict text")
	}
}

func TestExtractMonitoringSignals_NoConflictsDetectedDoesNotTriggerFailure(t *testing.T) {
	got := ExtractMonitoringSignals(MonitoringInputs{
		Attached:    1,
		Command:     "go",
		CommandArgs: "test ./internal/server",
		WorkingDir:  "/home/user/juli/internal/server",
		Excerpt:     "no conflicts detected\n",
	})

	if got.LooksLikeFailure {
		t.Fatalf("expected LooksLikeFailure to be false for 'no conflicts detected'")
	}
}

func TestExtractMonitoringSignals_BenignConfirmationDoesNotTriggerWait(t *testing.T) {
	got := ExtractMonitoringSignals(MonitoringInputs{
		Attached:    1,
		Command:     "npm",
		CommandArgs: "run build",
		WorkingDir:  "/home/user/juli/web",
		Excerpt:     "confirmation received from the service",
	})

	if got.LooksWaitingForInput {
		t.Fatalf("expected LooksWaitingForInput to be false for benign confirmation text")
	}
	if !got.HasRunningCommand {
		t.Fatalf("expected HasRunningCommand to remain true")
	}
}

func TestExtractMonitoringSignals_IncidentalYNDoesNotTriggerWait(t *testing.T) {
	got := ExtractMonitoringSignals(MonitoringInputs{
		Attached:    1,
		Command:     "npm",
		CommandArgs: "run build",
		WorkingDir:  "/home/user/juli/web",
		Excerpt:     "verification output [y/n] is shown in logs",
	})

	if got.LooksWaitingForInput {
		t.Fatalf("expected LooksWaitingForInput to be false for incidental y/n text")
	}
}

func TestBuildActivitySummary(t *testing.T) {
	tests := []struct {
		name string
		sig  MonitoringSignals
		want string
	}{
		{
			name: "running go tests in internal/server",
			sig: MonitoringSignals{
				State:        ActivityExecuting,
				CommandLabel: "Go tests",
				ContextLabel: "internal/server",
			},
			want: "Running Go tests in internal/server",
		},
		{
			name: "waiting for confirmation in codex",
			sig: MonitoringSignals{
				State:                ActivityWaiting,
				LooksWaitingForInput: true,
				ToolLabel:            "codex",
			},
			want: "Waiting for confirmation in codex",
		},
		{
			name: "waiting with unknown tool falls back cleanly",
			sig: MonitoringSignals{
				State:                ActivityWaiting,
				LooksWaitingForInput: true,
				ToolLabel:            "unknown",
			},
			want: "Waiting for input",
		},
		{
			name: "waiting with shell tool falls back cleanly",
			sig: MonitoringSignals{
				State:                ActivityWaiting,
				LooksWaitingForInput: true,
				ToolLabel:            "shell",
			},
			want: "Waiting for input",
		},
		{
			name: "build failed in web app",
			sig: MonitoringSignals{
				State:             ActivityNeedsAttention,
				LooksLikeBuildRun: true,
				ContextLabel:      "web app",
			},
			want: "Build failed in web app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildActivitySummary(tt.sig); got != tt.want {
				t.Fatalf("BuildActivitySummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLooksLikeFailure_StrongMarkersOnly(t *testing.T) {
	tests := []struct {
		name    string
		excerpt string
		want    bool
	}{
		{
			name:    "test failure marker",
			excerpt: "--- FAIL: TestLogin (0.00s)\nFAIL\n",
			want:    true,
		},
		{
			name:    "benign zero counts",
			excerpt: "0 errors\nerrors: 0\n0 failures\n",
			want:    false,
		},
		{
			name:    "plain success output",
			excerpt: "ok  github.com/cihann/juliterm/internal/server 0.010s\n",
			want:    false,
		},
		{
			name:    "no conflicts detected",
			excerpt: "no conflicts detected\n",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksLikeFailure(tt.excerpt); got != tt.want {
				t.Fatalf("looksLikeFailure() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractMonitoringSignals_GenericBuildingDoesNotTriggerBuildRun(t *testing.T) {
	got := ExtractMonitoringSignals(MonitoringInputs{
		Attached:   1,
		ToolType:   "codex",
		WorkingDir: "/home/user/juli",
		Excerpt:    "I am building a stronger understanding of this codebase.",
	})

	if got.LooksLikeBuildRun {
		t.Fatalf("expected LooksLikeBuildRun to be false for generic prose")
	}
	if got.HasAgentTool != true {
		t.Fatalf("expected HasAgentTool to remain true for agent activity")
	}
}

func TestLooksWaitingForInput_StrongPromptsOnly(t *testing.T) {
	tests := []struct {
		name    string
		excerpt string
		want    bool
	}{
		{
			name:    "press enter prompt",
			excerpt: "Press enter to confirm the next step",
			want:    true,
		},
		{
			name:    "waiting for confirmation prompt",
			excerpt: "Waiting for confirmation before continuing",
			want:    true,
		},
		{
			name:    "benign confirmation wording",
			excerpt: "confirmation received from the service",
			want:    false,
		},
		{
			name:    "benign confirmed wording",
			excerpt: "build confirmed successfully",
			want:    false,
		},
		{
			name:    "bare y/n bracket prompt",
			excerpt: "Apply changes? [y/n]",
			want:    true,
		},
		{
			name:    "bare y/n paren prompt",
			excerpt: "Proceed with this change? (y/n)",
			want:    true,
		},
		{
			name:    "aider style yes button",
			excerpt: "(Y)es/(N)o/(A)ll",
			want:    true,
		},
		{
			name:    "do you want to proceed",
			excerpt: "Do you want to proceed?",
			want:    true,
		},
		{
			name:    "do you want to continue",
			excerpt: "Do you want to continue?",
			want:    true,
		},
		{
			name:    "yes/no bracket prompt",
			excerpt: "Save file? [yes/no]",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := looksWaitingForInput(strings.ToLower(tt.excerpt)); got != tt.want {
				t.Fatalf("looksWaitingForInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeriveMonitoringResult_ExcerptStableAgentBecomesWaiting(t *testing.T) {
	input := MonitoringInputs{
		Attached:      1,
		ToolType:      "claude",
		Command:       "claude",
		WorkingDir:    "/home/user/project",
		Excerpt:       "What should I do next?",
		ExcerptStable: true,
	}

	got := DeriveMonitoringResult(input)
	if got.State != ActivityWaiting {
		t.Fatalf("State = %q, want %q", got.State, ActivityWaiting)
	}
	if got.Summary != "Waiting for confirmation in claude" {
		t.Fatalf("Summary = %q, want %q", got.Summary, "Waiting for confirmation in claude")
	}
}

func TestDeriveMonitoringResult_ExcerptStableNonAgentStaysIdle(t *testing.T) {
	input := MonitoringInputs{
		Attached:      1,
		ToolType:      "shell",
		Command:       "zsh",
		WorkingDir:    "/home/user/project",
		Excerpt:       "zsh$",
		ExcerptStable: true,
	}

	got := DeriveMonitoringResult(input)
	if got.State != ActivityIdle {
		t.Fatalf("State = %q, want %q (stable excerpt on non-agent should not force waiting)", got.State, ActivityIdle)
	}
}

func TestClassifyCommandLabel_CombinedCommandText(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		args string
		want string
	}{
		{
			name: "go test split across command and args",
			cmd:  "go",
			args: "test ./internal/server",
			want: "Go tests",
		},
		{
			name: "npm build split across command and args",
			cmd:  "npm",
			args: "run build",
			want: "web build",
		},
		{
			name: "full command in command field",
			cmd:  "go test ./internal/server",
			want: "Go tests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyCommandLabel(normalizedCommandText(tt.cmd, tt.args), tt.cmd, tt.args); got != tt.want {
				t.Fatalf("classifyCommandLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
