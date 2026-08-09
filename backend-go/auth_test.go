package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newAuthTestApp(t *testing.T, username, password string) *app {
	t.Helper()
	dataDir := t.TempDir()
	record, err := buildPasswordRecord(password)
	if err != nil {
		t.Fatal(err)
	}
	usersFile := filepath.Join(dataDir, "users.json")
	if err := writeJSONAtomic(usersFile, map[string]any{"users": map[string]any{username: record}}); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(dataDir, "dashboard-config.json")
	if err := os.WriteFile(configFile, []byte(`{"title":"Test","dashboards":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return &app{
		cfg: envConfig{
			DataDir: dataDir, ConfigFile: configFile, UsersFile: usersFile,
			SessionsFile:      filepath.Join(dataDir, "sessions.json"),
			SessionCookieName: sessionCookieNameDefault, SessionTTLSeconds: defaultSessionTTL,
		},
		sessions: map[string]sessionInfo{},
	}
}

func jsonRequest(t *testing.T, method, target string, payload map[string]any) *http.Request {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func TestSessionCookieSecurityAndLifetime(t *testing.T) {
	a := newAuthTestApp(t, "tester", "test-password")
	for _, testCase := range []struct {
		name   string
		https  bool
		secure bool
	}{
		{name: "http", secure: false},
		{name: "forwarded https", https: true, secure: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/login", nil)
			if testCase.https {
				request.Header.Set("X-Forwarded-Proto", "https")
			}
			response := httptest.NewRecorder()
			a.setSessionCookie(response, request, "token")
			cookies := response.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("cookies = %d, want 1", len(cookies))
			}
			cookie := cookies[0]
			if cookie.Secure != testCase.secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
				t.Fatalf("unexpected cookie flags: %#v", cookie)
			}
			if cookie.MaxAge != defaultSessionTTL || time.Until(cookie.Expires) < (9*365*24*time.Hour) {
				t.Fatalf("unexpected cookie lifetime: MaxAge=%d Expires=%v", cookie.MaxAge, cookie.Expires)
			}
		})
	}

	clearResponse := httptest.NewRecorder()
	a.clearSessionCookie(clearResponse, httptest.NewRequest(http.MethodPost, "/api/logout", nil))
	if cookie := clearResponse.Result().Cookies()[0]; cookie.MaxAge != -1 || !cookie.Expires.Before(time.Now()) {
		t.Fatalf("clear cookie was not expired: %#v", cookie)
	}
}

func TestUsernameChangeUpdatesAllSessions(t *testing.T) {
	a := newAuthTestApp(t, "tester", "test-password")
	expires := time.Now().Add(time.Hour).Unix()
	a.sessions["current"] = sessionInfo{Username: "tester", Expires: expires}
	a.sessions["other"] = sessionInfo{Username: "tester", Expires: expires}

	request := jsonRequest(t, http.MethodPost, "/api/auth/change-username", map[string]any{
		"currentPassword": "test-password", "newUsername": "renamed",
	})
	request.AddCookie(&http.Cookie{Name: sessionCookieNameDefault, Value: "current"})
	response := httptest.NewRecorder()
	a.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	for token, session := range a.sessions {
		if session.Username != "renamed" {
			t.Fatalf("session %s still uses %q", token, session.Username)
		}
	}
}

func TestPasswordChangeRevokesOtherSessions(t *testing.T) {
	a := newAuthTestApp(t, "tester", "old-password")
	expires := time.Now().Add(time.Hour).Unix()
	a.sessions["current"] = sessionInfo{Username: "tester", Expires: expires}
	a.sessions["other"] = sessionInfo{Username: "tester", Expires: expires}
	a.sessions["unrelated"] = sessionInfo{Username: "someone-else", Expires: expires}

	request := jsonRequest(t, http.MethodPost, "/api/auth/change-password", map[string]any{
		"currentPassword": "old-password", "newPassword": "new-password",
	})
	request.AddCookie(&http.Cookie{Name: sessionCookieNameDefault, Value: "current"})
	response := httptest.NewRecorder()
	a.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	if _, ok := a.sessions["other"]; ok {
		t.Fatal("other session was not revoked")
	}
	if _, ok := a.sessions["current"]; !ok {
		t.Fatal("current session was revoked")
	}
	if _, ok := a.sessions["unrelated"]; !ok {
		t.Fatal("unrelated session was revoked")
	}
}

func TestRemovedUserSessionIsRejected(t *testing.T) {
	a := newAuthTestApp(t, "tester", "test-password")
	a.sessions["ghost"] = sessionInfo{Username: "deleted-user", Expires: time.Now().Add(time.Hour).Unix()}
	request := jsonRequest(t, http.MethodPost, "/api/config", map[string]any{"config": map[string]any{}})
	request.AddCookie(&http.Cookie{Name: sessionCookieNameDefault, Value: "ghost"})
	response := httptest.NewRecorder()
	a.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestRemovedIconImportRoutesReturnNotFound(t *testing.T) {
	a := newAuthTestApp(t, "tester", "test-password")
	for _, target := range []string{"/api/icons/import-selfhst", "/api/icons/import-iconify"} {
		response := httptest.NewRecorder()
		a.ServeHTTP(response, jsonRequest(t, http.MethodPost, target, map[string]any{}))
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want %d", target, response.Code, http.StatusNotFound)
		}
	}
}
