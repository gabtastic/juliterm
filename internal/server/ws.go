package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	julipty "github.com/cihann/juliterm/internal/pty"
	"github.com/cihann/juliterm/internal/protocol"
	"github.com/coder/websocket"
)

// wsConn wraps a single WebSocket connection with its PTY session.
type wsConn struct {
	id            string
	conn          *websocket.Conn
	authSessionID string             // auth session ID from cookie (for revocation)
	sessionID     string             // tmux session this conn is attached to
	pty           *julipty.Session
	ptyCancel     context.CancelFunc // cancels forwardPTYOutput goroutine
	writeMu       sync.Mutex
}

// writeMessage sends a binary message over the WebSocket.
func (wc *wsConn) writeMessage(data []byte) error {
	wc.writeMu.Lock()
	defer wc.writeMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return wc.conn.Write(ctx, websocket.MessageBinary, data)
}

// wsAcceptOptions returns the WebSocket accept options with the current origin
// allowlist. When no extra origins are configured, it returns nil which makes
// coder/websocket enforce same-origin (Origin must match Host). When extra
// origins are set (e.g. a Cloudflare tunnel host), those are allowed alongside
// same-origin.
func (s *Server) wsAcceptOptions() *websocket.AcceptOptions {
	s.allowedOriginsMu.RLock()
	defer s.allowedOriginsMu.RUnlock()
	if len(s.allowedOrigins) == 0 {
		return nil
	}
	origins := make([]string, len(s.allowedOrigins))
	copy(origins, s.allowedOrigins)
	return &websocket.AcceptOptions{OriginPatterns: origins}
}

// handleWebSocket upgrades an HTTP request to a WebSocket connection,
// waits for a session selection, spawns a PTY, and bridges I/O.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	authID, ok := s.validateSession(r)
	if !ok {
		log.Printf("ws: rejected unauthorized connection")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, s.wsAcceptOptions())
	if err != nil {
		log.Printf("ws: accept failed: %v", err)
		return
	}

	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		conn.Close(websocket.StatusInternalError, "failed to generate connection id")
		return
	}

	wc := &wsConn{
		id:            hex.EncodeToString(idBytes),
		conn:          conn,
		authSessionID: authID,
	}

	s.fanout.RegisterConn(wc)
	log.Printf("ws[%s]: connected", wc.id)
	defer func() {
		log.Printf("ws[%s]: disconnected", wc.id)
		s.fanout.UnregisterConn(wc)
		if wc.ptyCancel != nil {
			wc.ptyCancel()
		}
		if wc.pty != nil {
			wc.pty.Close()
		}
		conn.CloseNow()
	}()

	// Send the initial session tree.
	if treeJSON, err := s.tmuxMgr.Tree().ToJSON(); err == nil {
		log.Printf("ws[%s]: sent initial session tree (%d sessions)", wc.id, len(s.tmuxMgr.Tree().GetSessions()))
		_ = wc.writeMessage(protocol.NewSessionTreeMessage(treeJSON))
	}

	// Read loop — handle messages from the browser.
	ctx := context.Background()
	s.readLoop(ctx, wc)
}

// readLoop reads binary messages from the WebSocket and dispatches them.
func (s *Server) readLoop(ctx context.Context, wc *wsConn) {
	for {
		_, data, err := wc.conn.Read(ctx)
		if err != nil {
			return
		}

		msgType, payload, err := protocol.ParseMessage(data)
		if err != nil {
			log.Printf("ws[%s]: parse error: %v", wc.id, err)
			continue
		}

		switch msgType {
		case protocol.MsgInput:
			s.handleInput(wc, payload)
		case protocol.MsgResize:
			s.handleResize(wc, payload)
		case protocol.MsgSessionTree:
			// Client sends session tree message to select/switch session.
			s.handleSessionSelect(wc, payload)
		case protocol.MsgSelectWindow:
			s.handleSelectWindow(wc, payload)
		default:
			log.Printf("ws[%s]: unknown message type 0x%02x", wc.id, msgType)
		}
	}
}

// handleSessionSelect processes a session selection from the client.
// The client sends MsgSessionTree with JSON {"sessionName": "..."}.
func (s *Server) handleSessionSelect(wc *wsConn, payload []byte) {
	var sel struct {
		SessionName string `json:"sessionName"`
	}
	if err := json.Unmarshal(payload, &sel); err != nil || sel.SessionName == "" {
		_ = wc.writeMessage(protocol.NewErrorMessage("bad_select", "invalid session selection"))
		return
	}

	// Cancel and close existing PTY if switching sessions.
	if wc.ptyCancel != nil {
		wc.ptyCancel()
	}
	if wc.pty != nil {
		wc.pty.Close()
		wc.pty = nil
		wc.sessionID = ""
	}

	// Spawn a PTY attached to the requested tmux session.
	// Default to 80x24 — the client will send a resize immediately after.
	log.Printf("ws[%s]: attaching to session %q", wc.id, sel.SessionName)
	ps, err := julipty.Attach(s.tmuxMgr.TmuxPath(), sel.SessionName, 80, 24)
	if err != nil {
		log.Printf("ws[%s]: pty attach to %q failed: %v", wc.id, sel.SessionName, err)
		_ = wc.writeMessage(protocol.NewErrorMessage("attach_error", err.Error()))
		return
	}
	log.Printf("ws[%s]: attached to session %q, streaming output", wc.id, sel.SessionName)

	ctx, cancel := context.WithCancel(context.Background())
	wc.pty = ps
	wc.ptyCancel = cancel
	wc.sessionID = sel.SessionName

	// Start forwarding PTY output to WebSocket.
	go s.forwardPTYOutput(ctx, wc, ps)
}

// handleSelectWindow switches the active window in the attached tmux session.
// The client sends MsgSelectWindow with JSON {"sessionName": "...", "windowIndex": N}.
func (s *Server) handleSelectWindow(wc *wsConn, payload []byte) {
	var sel struct {
		SessionName string `json:"sessionName"`
		WindowIndex int    `json:"windowIndex"`
	}
	if err := json.Unmarshal(payload, &sel); err != nil || sel.SessionName == "" {
		return
	}

	target := fmt.Sprintf("%s:%d", sel.SessionName, sel.WindowIndex)
	cmd := exec.Command(s.tmuxMgr.TmuxPath(), "select-window", "-t", target)
	if err := cmd.Run(); err != nil {
		log.Printf("ws[%s]: select-window %q failed: %v", wc.id, target, err)
	}
}

// forwardPTYOutput reads from the PTY and sends MsgOutput frames to the browser.
// It exits when the PTY read returns an error (fd closed) or the context is cancelled.
func (s *Server) forwardPTYOutput(ctx context.Context, wc *wsConn, ps *julipty.Session) {
	buf := make([]byte, 32*1024)
	for {
		n, err := ps.Read(buf)
		if n > 0 {
			msg := protocol.NewMessage(protocol.MsgOutput, buf[:n])
			if writeErr := wc.writeMessage(msg); writeErr != nil {
				return
			}
		}
		if err != nil {
			// Only log unexpected errors — not expected closes or EOF
			if err != io.EOF && ctx.Err() == nil {
				log.Printf("ws[%s]: pty read error: %v", wc.id, err)
			}
			// Only notify client if this wasn't an intentional close (session switch)
			if ctx.Err() == nil {
				_ = wc.writeMessage(protocol.NewErrorMessage("session_ended", "tmux session ended"))
			}
			return
		}
	}
}

// handleInput writes raw keyboard bytes directly to the PTY.
func (s *Server) handleInput(wc *wsConn, payload []byte) {
	if wc.pty == nil || len(payload) == 0 {
		return
	}
	if _, err := wc.pty.Write(payload); err != nil {
		log.Printf("ws[%s]: pty write error: %v", wc.id, err)
	}
}

// handleResize changes the PTY window size.
func (s *Server) handleResize(wc *wsConn, payload []byte) {
	if wc.pty == nil {
		return
	}
	var r protocol.ResizePayload
	if err := json.Unmarshal(payload, &r); err != nil {
		return
	}
	if r.Cols <= 0 || r.Rows <= 0 {
		return
	}
	if err := wc.pty.Resize(uint16(r.Cols), uint16(r.Rows)); err != nil {
		log.Printf("ws[%s]: pty resize to %dx%d: %v", wc.id, r.Cols, r.Rows, err)
	}
}
