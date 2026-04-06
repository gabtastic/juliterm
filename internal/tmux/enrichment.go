package tmux

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ProcessInfo holds the comm (basename) and full args of a child process.
type ProcessInfo struct {
	Comm string
	Args string
}

// ProcessSnapshot maps parent PID → first child process info.
// Built once per enrichment tick via SnapshotProcesses, then used for O(1) lookups.
type ProcessSnapshot map[int]ProcessInfo

// SnapshotProcesses runs `ps -ax` once and builds a lookup table keyed by
// parent PID. Only the first child found per parent is stored.
func SnapshotProcesses(ctx context.Context) ProcessSnapshot {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ps", "-ax", "-o", "ppid=,comm=,args=").Output()
	if err != nil {
		return nil
	}

	snapshot := make(ProcessSnapshot)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ppid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		// Only store the first child per parent.
		if _, exists := snapshot[ppid]; exists {
			continue
		}
		args := ""
		if len(fields) >= 3 {
			args = strings.Join(fields[2:], " ")
		}
		snapshot[ppid] = ProcessInfo{Comm: fields[1], Args: args}
	}
	return snapshot
}

// DetectToolType identifies the agentic coding tool running in a pane by
// looking up the first child process of the shell (panePID) in a pre-built
// process snapshot. Returns one of:
// "claude", "codex", "opencode", "aider", "cursor", "shell", "unknown".
func DetectToolType(snapshot ProcessSnapshot, panePID int) string {
	if panePID <= 0 {
		return "unknown"
	}

	child, ok := snapshot[panePID]
	if !ok {
		return "shell" // no child → idle shell
	}

	// Exact comm matches.
	switch child.Comm {
	case "claude":
		return "claude"
	case "opencode":
		return "opencode"
	case "aider":
		return "aider"
	case "cursor":
		return "cursor"
	}

	// Args-based detection for tools that run as node/python subprocesses.
	if strings.Contains(child.Args, "/codex") || strings.Contains(child.Args, " codex") {
		return "codex"
	}
	if strings.Contains(child.Args, "aider") {
		return "aider"
	}

	return "unknown"
}

// GetGitInfo extracts the current branch and "owner/repo" from a working directory.
// Returns empty strings (without error) if the directory is not a git repo.
func GetGitInfo(ctx context.Context, workingDir string) (branch, repo string) {
	if workingDir == "" {
		return "", ""
	}

	// Branch name.
	branchCtx, branchCancel := context.WithTimeout(ctx, 2*time.Second)
	defer branchCancel()
	branchOut, err := exec.CommandContext(branchCtx, "git", "-C", workingDir,
		"rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", ""
	}
	branch = strings.TrimSpace(string(branchOut))

	// Remote URL → owner/repo.
	remoteCtx, remoteCancel := context.WithTimeout(ctx, 2*time.Second)
	defer remoteCancel()
	remoteOut, err := exec.CommandContext(remoteCtx, "git", "-C", workingDir,
		"remote", "get-url", "origin").Output()
	if err != nil {
		return branch, ""
	}
	repo = ParseGitRemoteURL(strings.TrimSpace(string(remoteOut)))

	return branch, repo
}

var (
	// git@github.com:owner/repo.git or git@github.com:owner/repo
	sshRemoteRe = regexp.MustCompile(`[^:]+:([^/]+/[^/]+?)(?:\.git)?$`)
	// https://github.com/owner/repo.git or https://github.com/owner/repo
	httpsRemoteRe = regexp.MustCompile(`https?://[^/]+/([^/]+/[^/]+?)(?:\.git)?$`)
)

// ParseGitRemoteURL extracts "owner/repo" from an SSH or HTTPS git remote URL.
// Returns empty string if the URL cannot be parsed.
func ParseGitRemoteURL(remoteURL string) string {
	if m := httpsRemoteRe.FindStringSubmatch(remoteURL); len(m) > 1 {
		return m[1]
	}
	if m := sshRemoteRe.FindStringSubmatch(remoteURL); len(m) > 1 {
		return m[1]
	}
	return ""
}

// DeriveSessionStatus computes a human-readable status from tmux metadata.
func DeriveSessionStatus(attached int, activePaneDead bool, hasChildProcess bool) string {
	if attached == 0 {
		return "detached"
	}
	if activePaneDead {
		return "error"
	}
	if !hasChildProcess {
		return "idle"
	}
	return "running"
}
