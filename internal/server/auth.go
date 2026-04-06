package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	cookieName   = "juli_session"
	cookieMaxAge = 86400 * 7 // 7 days
)

// loginRequest is the JSON body for POST /api/login.
type loginRequest struct {
	Token string `json:"token"`
}

// loginResponse is returned on successful authentication.
type loginResponse struct {
	OK bool `json:"ok"`
}

// handleLogin accepts JSON {"token": "..."}, validates it against the server's
// shared token, and sets an HTTP-only session cookie on success.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if subtle.ConstantTimeCompare([]byte(req.Token), []byte(s.token)) != 1 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid token"})
		return
	}

	// Generate a signed session cookie and register it in the session store.
	sessionID := generateSessionID()
	s.sessions.Add(sessionID)
	signed := sessionID + "." + signSession(sessionID, s.sessionSecret)

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    signed,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{OK: true})
}

// authMiddleware validates the session cookie on every request. If the cookie
// is missing or invalid the request is rejected with 401.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.validateSession(r); !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handleLogout removes the session from the server-side store, clears the
// cookie, and closes any associated WebSocket connections.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract session ID from cookie and revoke it.
	if id, ok := extractSessionID(r, s.sessionSecret); ok {
		s.sessions.Delete(id)
		s.fanout.CloseByAuthSession(id)
	}

	// Clear the cookie regardless.
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResponse{OK: true})
}

// validateSession checks whether the request carries a valid session cookie
// and the session exists in the server-side store. It returns the auth session
// ID on success so callers (e.g. WebSocket upgrade) can use it without
// re-parsing the cookie.
func (s *Server) validateSession(r *http.Request) (string, bool) {
	id, ok := extractSessionID(r, s.sessionSecret)
	if !ok {
		return "", false
	}
	if s.sessions.Get(id) == nil {
		return "", false
	}
	return id, true
}

// extractSessionID parses the session cookie, verifies the HMAC signature,
// and returns the raw session ID. It returns ("", false) if the cookie is
// missing, malformed, or has an invalid signature.
func extractSessionID(r *http.Request, secret []byte) (string, bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", false
	}
	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	sessionID, sig := parts[0], parts[1]
	expected := signSession(sessionID, secret)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return "", false
	}
	return sessionID, true
}

// generateSessionID returns a random 32-byte hex string.
func generateSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Catastrophic — crypto/rand should never fail.
		panic("juli: crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// signSession computes an HMAC-SHA256 of the session ID using the given secret
// and returns the hex-encoded signature.
func signSession(id string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(id))
	return hex.EncodeToString(mac.Sum(nil))
}

