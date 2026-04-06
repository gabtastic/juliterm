// Package tunnel provides Cloudflare Quick Tunnel integration for Juli.
// It spawns cloudflared as a child process and extracts the tunnel URL.
package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"time"
)

// urlPattern matches the Quick Tunnel URL in cloudflared's stderr output.
var urlPattern = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// Start spawns a Cloudflare Quick Tunnel via cloudflared and blocks until a
// tunnel URL is available (up to 15 seconds). Returns the URL and a stop
// function that terminates cloudflared. The stop function is safe to call
// even if Start returned an error (no-op).
//
// Start is non-fatal by design: if cloudflared is missing, exits immediately,
// or times out, the caller should log the error and continue serving locally.
func Start(ctx context.Context, port int) (url string, stop func(), err error) {
	noop := func() {}

	// Check if cloudflared is installed.
	cloudflaredPath, err := exec.LookPath("cloudflared")
	if err != nil {
		return "", noop, fmt.Errorf("cloudflared not found in PATH. Install it:\n" +
			"  macOS:  brew install cloudflared\n" +
			"  Linux:  apt install cloudflared (or see https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/)\n" +
			"  Other:  https://github.com/cloudflare/cloudflared/releases")
	}

	// Child context so we can cancel cloudflared independently of the parent.
	childCtx, cancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(childCtx, cloudflaredPath, "tunnel", "--url",
		fmt.Sprintf("http://localhost:%d", port))

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return "", noop, fmt.Errorf("cloudflared stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return "", noop, fmt.Errorf("cloudflared start: %w", err)
	}

	// Channel receives the first URL found in stderr.
	urlCh := make(chan string, 1)

	// Goroutine scans stderr for the tunnel URL and continues draining after.
	go func() {
		scanner := bufio.NewScanner(stderr)
		found := false
		for scanner.Scan() {
			line := scanner.Text()
			if !found {
				if m := urlPattern.FindString(line); m != "" {
					urlCh <- m
					found = true
				}
			}
		}
		// If cloudflared exits without ever producing a URL, signal that.
		if !found {
			urlCh <- ""
		}
	}()

	// Wait for URL, process exit, or timeout.
	select {
	case u := <-urlCh:
		if u == "" {
			cancel()
			// Wait briefly to collect the exit error for a better message.
			_ = cmd.Wait()
			return "", noop, fmt.Errorf("cloudflared exited without producing a tunnel URL")
		}

		// Monitor for unexpected crash after URL is available.
		go func() {
			if err := cmd.Wait(); err != nil && childCtx.Err() == nil {
				log.Printf("WARNING: cloudflared exited unexpectedly: %v", err)
				log.Printf("Tunnel is no longer active. Juli continues serving locally.")
			}
		}()

		return u, cancel, nil

	case <-time.After(15 * time.Second):
		cancel()
		_ = cmd.Wait()
		return "", noop, fmt.Errorf("cloudflared did not produce a tunnel URL within 15 seconds")

	case <-ctx.Done():
		cancel()
		_ = cmd.Wait()
		return "", noop, ctx.Err()
	}
}

// ParseURL extracts a trycloudflare.com URL from a line of text.
// Exported for testing. Returns "" if no match.
func ParseURL(line string) string {
	return urlPattern.FindString(line)
}