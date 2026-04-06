package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// helper: perform a login and return the Set-Cookie value.
func doLogin(t *testing.T, s *Server, token string) string {
	t.Helper()
	body := `{"token":"` + token + `"}`
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleLogin(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login: status = %d, want %d", w.Code, http.StatusOK)
	}
	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == cookieName {
			return c.Value
		}
	}
	t.Fatal("login: no session cookie set")
	return ""
}

// helper: create an HTTP request with the session cookie attached.
func reqWithCookie(method, path, cookie string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: cookie})
	return req
}

func TestLogin_RegistersSessionInStore(t *testing.T) {
	mgr := newMockManager()
	s := newTestServer(mgr)

	cookie := doLogin(t, s, "test")

	// The session ID is the part before the dot.
	parts := strings.SplitN(cookie, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("cookie format: got %q, want <id>.<sig>", cookie)
	}
	sessionID := parts[0]

	if sess := s.sessions.Get(sessionID); sess == nil {
		t.Error("session not found in store after login")
	}
}

func TestValidateSession_ValidCookieAndStore(t *testing.T) {
	mgr := newMockManager()
	s := newTestServer(mgr)

	cookie := doLogin(t, s, "test")
	req := reqWithCookie("GET", "/api/sessions", cookie)

	if _, ok := s.validateSession(req); !ok {
		t.Error("validateSession returned false for valid cookie + store entry")
	}
}

func TestValidateSession_ValidHMAC_MissingStore(t *testing.T) {
	mgr := newMockManager()
	s := newTestServer(mgr)

	cookie := doLogin(t, s, "test")

	// Remove the session from the store but keep the cookie valid.
	parts := strings.SplitN(cookie, ".", 2)
	s.sessions.Delete(parts[0])

	req := reqWithCookie("GET", "/api/sessions", cookie)
	if _, ok := s.validateSession(req); ok {
		t.Error("validateSession returned true for valid HMAC but missing store entry")
	}
}

func TestValidateSession_ExpiredTTL(t *testing.T) {
	mgr := newMockManager()
	s := newTestServer(mgr)
	// Override the store with a very short TTL.
	s.sessions = NewSessionStore(1 * time.Millisecond)

	cookie := doLogin(t, s, "test")
	time.Sleep(5 * time.Millisecond) // let it expire

	req := reqWithCookie("GET", "/api/sessions", cookie)
	if _, ok := s.validateSession(req); ok {
		t.Error("validateSession returned true for expired session")
	}
}

func TestValidateSession_NoCookie(t *testing.T) {
	mgr := newMockManager()
	s := newTestServer(mgr)

	req := httptest.NewRequest("GET", "/api/sessions", nil)
	if _, ok := s.validateSession(req); ok {
		t.Error("validateSession returned true with no cookie")
	}
}

func TestValidateSession_BadSignature(t *testing.T) {
	mgr := newMockManager()
	s := newTestServer(mgr)

	req := reqWithCookie("GET", "/api/sessions", "fakeid.fakesig")
	if _, ok := s.validateSession(req); ok {
		t.Error("validateSession returned true for invalid signature")
	}
}

func TestLogout_RemovesSessionAndClearsCookie(t *testing.T) {
	mgr := newMockManager()
	s := newTestServer(mgr)

	cookie := doLogin(t, s, "test")
	parts := strings.SplitN(cookie, ".", 2)
	sessionID := parts[0]

	req := reqWithCookie("POST", "/api/logout", cookie)
	w := httptest.NewRecorder()
	s.handleLogout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("logout: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Session must be gone from the store.
	if s.sessions.Get(sessionID) != nil {
		t.Error("session still in store after logout")
	}

	// Cookie must be cleared (MaxAge < 0).
	for _, c := range w.Result().Cookies() {
		if c.Name == cookieName && c.MaxAge >= 0 {
			t.Errorf("cookie MaxAge = %d, want < 0", c.MaxAge)
		}
	}
}

func TestLogout_Idempotent(t *testing.T) {
	mgr := newMockManager()
	s := newTestServer(mgr)

	cookie := doLogin(t, s, "test")

	// Logout twice — second should still return 200.
	for i := 0; i < 2; i++ {
		req := reqWithCookie("POST", "/api/logout", cookie)
		w := httptest.NewRecorder()
		s.handleLogout(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("logout attempt %d: status = %d, want %d", i+1, w.Code, http.StatusOK)
		}
	}
}

func TestLogout_NoCookie(t *testing.T) {
	mgr := newMockManager()
	s := newTestServer(mgr)

	// Logout without a cookie should still return 200 (idempotent).
	req := httptest.NewRequest("POST", "/api/logout", nil)
	w := httptest.NewRecorder()
	s.handleLogout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("logout without cookie: status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestLogin_InvalidToken(t *testing.T) {
	mgr := newMockManager()
	s := newTestServer(mgr)

	body := `{"token":"wrong"}`
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.handleLogin(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("login with wrong token: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Error("login with wrong token: missing error field in response")
	}
}

func TestAuthMiddleware_RejectsRevokedSession(t *testing.T) {
	mgr := newMockManager()
	s := newTestServer(mgr)

	cookie := doLogin(t, s, "test")
	parts := strings.SplitN(cookie, ".", 2)

	// Revoke the session.
	s.sessions.Delete(parts[0])

	// Try to access a protected endpoint.
	req := reqWithCookie("GET", "/api/sessions", cookie)
	w := httptest.NewRecorder()
	handler := s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("revoked session: status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
