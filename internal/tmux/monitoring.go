package tmux

import (
	"path/filepath"
	"strings"
	"time"
)

type ActivityState string

const (
	ActivityNeedsAttention ActivityState = "needs_attention"
	ActivityExecuting      ActivityState = "executing"
	ActivityCoding         ActivityState = "coding"
	ActivityWaiting        ActivityState = "waiting"
	ActivityIdle           ActivityState = "idle"
	ActivityDetached       ActivityState = "detached"
	ActivityUnknown        ActivityState = "unknown"
)

type MonitoringInputs struct {
	Attached      int
	ToolType      string
	Command       string
	CommandArgs   string
	WorkingDir    string
	GitRepo       string
	GitBranch     string
	LastActivity  int64
	Now           time.Time
	Excerpt       string
	ExcerptStable bool // true when excerpt has not changed for ≥2 polling cycles
}

type MonitoringSignals struct {
	Attached             int
	State                ActivityState
	HasAgentTool         bool
	HasRunningCommand    bool
	LooksLikeTestRun     bool
	LooksLikeBuildRun    bool
	LooksLikeInstallRun  bool
	LooksWaitingForInput bool
	LooksLikeFailure     bool
	LooksLikePromptIdle  bool
	LooksExcerptStable   bool
	CommandLabel         string
	ContextLabel         string
	ToolLabel            string
}

type MonitoringResult struct {
	State   ActivityState
	Summary string
}

func ExtractMonitoringSignals(in MonitoringInputs) MonitoringSignals {
	excerpt := strings.ToLower(normalizeMonitoringExcerpt(in.Excerpt))
	latestLine := latestMonitoringLine(excerpt)
	commandText := normalizedCommandText(in.Command, in.CommandArgs)
	toolType := strings.ToLower(strings.TrimSpace(in.ToolType))
	waiting := looksWaitingForInput(latestLine)
	promptIdle := strings.HasSuffix(latestLine, "$") || strings.HasSuffix(latestLine, "%")
	agentTool := isAgentToolType(toolType)
	runningCommand := commandText != "" && !isShellCommand(commandText) && !waiting && !isAgentLauncherCommand(commandText, toolType)

	contextLabel := deriveMonitoringContextLabel(in.WorkingDir, in.GitRepo)

	return MonitoringSignals{
		Attached:             in.Attached,
		HasAgentTool:         agentTool && !waiting,
		HasRunningCommand:    runningCommand,
		LooksLikeTestRun:     strings.Contains(commandText, "go test") || (runningCommand && !promptIdle && strings.Contains(excerpt, "=== run")),
		LooksLikeBuildRun:    strings.Contains(commandText, "go build") || strings.Contains(commandText, "npm run build"),
		LooksLikeInstallRun:  strings.Contains(commandText, "go install") || strings.Contains(commandText, "npm install") || strings.Contains(commandText, "npm ci"),
		LooksWaitingForInput: waiting,
		LooksLikeFailure:     looksLikeFailure(latestLine),
		LooksLikePromptIdle:  promptIdle,
		LooksExcerptStable:   in.ExcerptStable && agentTool,
		CommandLabel:         classifyCommandLabel(commandText, in.Command, in.CommandArgs),
		ContextLabel:         contextLabel,
		ToolLabel:            strings.TrimSpace(in.ToolType),
	}
}

func DeriveActivityState(sig MonitoringSignals) ActivityState {
	switch {
	case sig.LooksLikeFailure:
		return ActivityNeedsAttention
	case sig.HasRunningCommand || sig.LooksLikeTestRun || sig.LooksLikeBuildRun || sig.LooksLikeInstallRun:
		return ActivityExecuting
	case sig.HasAgentTool && sig.LooksExcerptStable:
		return ActivityWaiting
	case sig.HasAgentTool:
		return ActivityCoding
	case sig.LooksWaitingForInput:
		return ActivityWaiting
	case sig.Attached == 0:
		return ActivityDetached
	case sig.LooksLikePromptIdle:
		return ActivityIdle
	default:
		return ActivityUnknown
	}
}

func BuildActivitySummary(sig MonitoringSignals) string {
	contextLabel := summarizeMonitoringContextLabel(sig)
	switch sig.State {
	case ActivityNeedsAttention:
		prefix := "Session needs attention"
		switch {
		case sig.LooksLikeTestRun:
			prefix = "Test failed"
		case sig.LooksLikeBuildRun:
			prefix = "Build failed"
		case sig.LooksLikeInstallRun:
			prefix = "Install failed"
		case sig.CommandLabel != "":
			prefix = "Command failed"
		}
		if contextLabel != "" {
			return prefix + " in " + contextLabel
		}
		return prefix
	case ActivityExecuting:
		if sig.CommandLabel != "" && contextLabel != "" {
			return "Running " + sig.CommandLabel + " in " + contextLabel
		}
		if sig.CommandLabel != "" {
			return "Running " + sig.CommandLabel
		}
		return "Executing command"
	case ActivityCoding:
		if contextLabel != "" {
			return "Editing files in " + contextLabel
		}
		return "Coding"
	case ActivityWaiting:
		if label := waitingToolLabel(sig.ToolLabel); label != "" {
			return "Waiting for confirmation in " + label
		}
		return "Waiting for input"
	case ActivityIdle:
		return "Idle at shell prompt"
	case ActivityDetached:
		return "No attached clients"
	default:
		return ""
	}
}

func DeriveMonitoringResult(in MonitoringInputs) MonitoringResult {
	signals := ExtractMonitoringSignals(in)
	signals.State = DeriveActivityState(signals)
	return MonitoringResult{
		State:   signals.State,
		Summary: BuildActivitySummary(signals),
	}
}

func normalizeMonitoringExcerpt(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSpace(s)
}

func latestMonitoringLine(excerpt string) string {
	lines := strings.Split(normalizeMonitoringExcerpt(excerpt), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func normalizedCommandText(cmd, args string) string {
	fields := strings.Fields(strings.TrimSpace(strings.Join([]string{cmd, args}, " ")))
	return strings.ToLower(strings.Join(fields, " "))
}

func deriveMonitoringContextLabel(workingDir, gitRepo string) string {
	if workingDir != "" {
		if label := contextLabelFromWorkingDir(workingDir); label != "" {
			return label
		}
	}
	return strings.TrimSpace(gitRepo)
}

func contextLabelFromWorkingDir(workingDir string) string {
	cleaned := filepath.Clean(strings.TrimSpace(workingDir))
	if cleaned == "" || cleaned == "." || cleaned == string(filepath.Separator) {
		return ""
	}

	parts := strings.Split(cleaned, string(filepath.Separator))
	if len(parts) == 0 {
		return ""
	}

	base := parts[len(parts)-1]
	if len(parts) >= 2 {
		parent := parts[len(parts)-2]
		switch parent {
		case "internal", "cmd", "pkg":
			return filepath.Join(parent, base)
		}
	}

	return base
}

func summarizeMonitoringContextLabel(sig MonitoringSignals) string {
	if sig.ContextLabel == "" {
		return ""
	}
	if sig.ContextLabel == "web" && (sig.LooksLikeBuildRun || sig.LooksLikeInstallRun) {
		return "web app"
	}
	return sig.ContextLabel
}

func isAgentToolType(toolType string) bool {
	switch toolType {
	case "", "shell", "unknown":
		return false
	default:
		return true
	}
}

func waitingToolLabel(toolLabel string) string {
	switch strings.ToLower(strings.TrimSpace(toolLabel)) {
	case "", "shell", "unknown":
		return ""
	default:
		return strings.TrimSpace(toolLabel)
	}
}

func isShellCommand(commandText string) bool {
	first := firstCommandToken(commandText)
	switch first {
	case "zsh", "bash", "fish", "sh":
		return true
	default:
		return false
	}
}

func isAgentLauncherCommand(commandText, toolType string) bool {
	if !isAgentToolType(toolType) {
		return false
	}
	fields := strings.Fields(strings.TrimSpace(commandText))
	if len(fields) == 0 {
		return false
	}
	if fields[0] == toolType {
		return true
	}
	// Claude Code (and similar tools) set their process title to their version
	// number (e.g. "2.1.87"), which tmux reports as pane_current_command.
	if looksLikeVersionString(fields[0]) {
		return true
	}
	if !isLauncherCommand(fields[0]) {
		return false
	}
	return strings.Contains(strings.Join(fields[1:], " "), toolType)
}

// looksLikeVersionString returns true for semver-like strings (e.g. "2.1.87",
// "1.0.0-beta"). Agent tools sometimes use their version as the process title.
func looksLikeVersionString(s string) bool {
	if s == "" {
		return false
	}
	// Must start with a digit and contain at least one dot.
	if s[0] < '0' || s[0] > '9' {
		return false
	}
	hasDot := false
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9', c == '.', c == '-', c == '+':
			if c == '.' {
				hasDot = true
			}
		default:
			return false
		}
	}
	return hasDot
}

func isLauncherCommand(command string) bool {
	switch command {
	case "node", "python", "python3", "uv", "uvx":
		return true
	default:
		return false
	}
}

func looksWaitingForInput(excerpt string) bool {
	for _, needle := range []string{
		"press enter to confirm",
		"press enter",
		"press return",
		"waiting for confirmation",
		"confirm the next step",
		"hit enter",
		"are you sure? [y/n]",
		"continue? [y/n]",
		"press y to continue",
		"(y)es",
	} {
		if strings.Contains(excerpt, needle) {
			return true
		}
	}
	// Position-sensitive: prompt indicators at end of line are strong signals,
	// but in the middle of a sentence they're incidental text.
	for _, suffix := range []string{
		"[y/n]",
		"(y/n)",
		"[yes/no]",
		"(yes/no)",
		"do you want to proceed?",
		"do you want to continue?",
	} {
		if strings.HasSuffix(excerpt, suffix) {
			return true
		}
	}
	return false
}

func looksLikeFailure(excerpt string) bool {
	for _, line := range strings.Split(normalizeMonitoringExcerpt(excerpt), "\n") {
		line = strings.ToLower(strings.TrimSpace(line))
		if line == "" {
			continue
		}
		switch {
		case line == "fail", line == "failed", line == "error", line == "fatal", line == "panic":
			return true
		case strings.HasPrefix(line, "--- fail:"):
			return true
		case strings.HasPrefix(line, "fail:"):
			return true
		case strings.HasPrefix(line, "failed:"):
			return true
		case strings.HasPrefix(line, "error:"):
			return true
		case strings.HasPrefix(line, "fatal:"):
			return true
		case strings.HasPrefix(line, "panic:"):
			return true
		case strings.Contains(line, "exit status ") && !strings.Contains(line, "exit status 0"):
			return true
		case strings.Contains(line, "failed to compile"), strings.Contains(line, "failed to build"), strings.Contains(line, "compilation failed"):
			return true
		case strings.Contains(line, "build failed"), strings.Contains(line, "test failed"), strings.Contains(line, "command failed"):
			return true
		case strings.Contains(line, "merge conflict"), strings.Contains(line, "conflict failed"), strings.Contains(line, "failed due to conflict"):
			return true
		case strings.Contains(line, "conflicts detected") && !strings.Contains(line, "no conflicts detected"):
			return true
		}
	}
	return false
}

func firstCommandToken(commandText string) string {
	fields := strings.Fields(strings.TrimSpace(commandText))
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func classifyCommandLabel(commandText, cmd, args string) string {
	switch {
	case strings.Contains(commandText, "go test"):
		return "Go tests"
	case strings.Contains(commandText, "go build"):
		return "Go build"
	case strings.Contains(commandText, "npm run build"):
		return "web build"
	case strings.Contains(commandText, "npm install") || strings.Contains(commandText, "npm ci"):
		return "web dependencies"
	case cmd != "":
		return strings.TrimSpace(cmd)
	case args != "":
		return strings.TrimSpace(args)
	case commandText != "":
		return firstCommandToken(commandText)
	default:
		return ""
	}
}
