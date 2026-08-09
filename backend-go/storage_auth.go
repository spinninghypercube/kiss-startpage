package main

import (
	"bytes"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (a *app) ensureFilesReady() error {
	if err := os.MkdirAll(a.cfg.DataDir, 0o750); err != nil {
		return err
	}
	a.fileMu.Lock()
	defer a.fileMu.Unlock()

	if _, err := os.Stat(a.cfg.ConfigFile); errors.Is(err, os.ErrNotExist) {
		cfg, ok := readJSONAny(a.cfg.DefaultConfigPath)
		if !ok {
			cfg = map[string]any{
				"title": "KISS Startpage",
				"dashboards": []any{
					map[string]any{
						"id":     "dashboard-1",
						"label":  "Startpage 1",
						"groups": []any{},
					},
				},
			}
		}
		if err := writeJSONAtomic(a.cfg.ConfigFile, normalizeConfig(cfg)); err != nil {
			return err
		}
	}

	if _, err := os.Stat(a.cfg.UsersFile); errors.Is(err, os.ErrNotExist) {
		if err := writeJSONAtomic(a.cfg.UsersFile, map[string]any{"users": map[string]any{}}); err != nil {
			return err
		}
	}
	return nil
}

func readJSONAny(path string) (any, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, false
	}
	return v, true
}

func writeJSONAtomic(path string, payload any) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

func normalizeConfig(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]any{
			"title":        "KISS Startpage",
			"themePresets": []any{},
			"dashboards":   []any{},
		}
	}
	out := map[string]any{}
	for k, val := range m {
		out[k] = val
	}
	title := strings.TrimSpace(asString(out["title"]))
	if title == "" {
		title = "KISS Startpage"
	}
	out["title"] = title
	if _, ok := out["themePresets"].([]any); !ok {
		if out["themePresets"] == nil {
			out["themePresets"] = []any{}
		}
	}
	if _, ok := out["dashboards"].([]any); !ok {
		out["dashboards"] = []any{}
	}
	return out
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (a *app) getUsersPayload() map[string]any {
	a.fileMu.Lock()
	defer a.fileMu.Unlock()
	v, ok := readJSONAny(a.cfg.UsersFile)
	if !ok {
		return map[string]any{"users": map[string]any{}}
	}
	m, ok := v.(map[string]any)
	if !ok {
		return map[string]any{"users": map[string]any{}}
	}
	if _, ok := m["users"].(map[string]any); !ok {
		m["users"] = map[string]any{}
	}
	return m
}

func (a *app) updateUsersPayload(update func(map[string]any) error) error {
	a.fileMu.Lock()
	defer a.fileMu.Unlock()
	v, ok := readJSONAny(a.cfg.UsersFile)
	if !ok {
		return fmt.Errorf("failed to read users")
	}
	payload, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid users payload")
	}
	if _, ok := payload["users"].(map[string]any); !ok {
		payload["users"] = map[string]any{}
	}
	if err := update(payload); err != nil {
		return err
	}
	return writeJSONAtomic(a.cfg.UsersFile, payload)
}

func (a *app) getConfigPayload() map[string]any {
	a.fileMu.Lock()
	defer a.fileMu.Unlock()
	v, ok := readJSONAny(a.cfg.ConfigFile)
	if !ok {
		return normalizeConfig(nil)
	}
	return normalizeConfig(v)
}

func (a *app) saveConfigPayload(cfg any) (map[string]any, error) {
	norm := normalizeConfig(cfg)
	a.fileMu.Lock()
	defer a.fileMu.Unlock()
	if err := writeJSONAtomic(a.cfg.ConfigFile, norm); err != nil {
		return nil, err
	}
	return norm, nil
}

func usersMap(payload map[string]any) map[string]any {
	users, _ := payload["users"].(map[string]any)
	if users == nil {
		users = map[string]any{}
		payload["users"] = users
	}
	return users
}

func hasAnyUsers(payload map[string]any) bool {
	for k := range usersMap(payload) {
		if strings.TrimSpace(k) != "" {
			return true
		}
	}
	return false
}

func validUsername(username string) bool {
	return validUsernameRe.MatchString(username)
}

func hashPassword(password, saltHex string, iterations int) (string, error) {
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return "", err
	}
	digest, err := pbkdf2.Key(sha256.New, password, salt, iterations, 32)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(digest), nil
}

func buildPasswordRecord(password string) (map[string]any, error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return nil, err
	}
	saltHex := hex.EncodeToString(saltBytes)
	iterations := 210000
	hashed, err := hashPassword(password, saltHex, iterations)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"salt":       saltHex,
		"iterations": iterations,
		"hash":       hashed,
	}, nil
}

func verifyPassword(password string, record any) bool {
	m, ok := record.(map[string]any)
	if !ok {
		return false
	}
	salt, _ := m["salt"].(string)
	hashStr, _ := m["hash"].(string)
	iter := asInt(m["iterations"], 210000)
	if salt == "" || hashStr == "" {
		return false
	}
	calculated, err := hashPassword(password, salt, iter)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(calculated), []byte(hashStr)) == 1
}

func userRequiresPasswordChange(payload map[string]any, username string) bool {
	rec, ok := usersMap(payload)[username].(map[string]any)
	if !ok {
		return false
	}
	v, _ := rec["mustChangePassword"].(bool)
	return v
}

func (a *app) isPasswordChangeRequired(username string) bool {
	return userRequiresPasswordChange(a.getUsersPayload(), username)
}

func clearPasswordChangeRequired(payload map[string]any, username string) bool {
	rec, ok := usersMap(payload)[username].(map[string]any)
	if !ok {
		return false
	}
	v, _ := rec["mustChangePassword"].(bool)
	if !v {
		return false
	}
	rec["mustChangePassword"] = false
	usersMap(payload)[username] = rec
	return true
}

func asInt(v any, fallback int) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		n, err := t.Int64()
		if err == nil {
			return int(n)
		}
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err == nil {
			return n
		}
	}
	return fallback
}

func (a *app) loadSessions() {
	raw, err := os.ReadFile(a.cfg.SessionsFile)
	if err != nil {
		return // file doesn't exist yet, start fresh
	}
	var sessions map[string]sessionInfo
	if err := json.Unmarshal(raw, &sessions); err != nil {
		log.Printf("warn: could not parse sessions file: %v", err)
		return
	}
	a.sessMu.Lock()
	defer a.sessMu.Unlock()
	now := time.Now().Unix()
	for token, s := range sessions {
		if s.Expires > now {
			a.sessions[token] = s
		}
	}
}

func (a *app) saveSessionsLocked() {
	if err := writeJSONAtomic(a.cfg.SessionsFile, a.sessions); err != nil {
		log.Printf("warn: could not save sessions: %v", err)
	}
}

func (a *app) pruneSessionsLocked() {
	now := time.Now().Unix()
	for token, s := range a.sessions {
		if s.Expires <= now {
			delete(a.sessions, token)
		}
	}
}

func (a *app) createSession(username string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	a.sessMu.Lock()
	defer a.sessMu.Unlock()
	a.pruneSessionsLocked()
	a.sessions[token] = sessionInfo{
		Username: username,
		Expires:  time.Now().Unix() + int64(a.cfg.SessionTTLSeconds),
	}
	a.saveSessionsLocked()
	return token, nil
}

func (a *app) resolveSessionUsername(r *http.Request) string {
	cookie, err := r.Cookie(a.cfg.SessionCookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return ""
	}
	a.sessMu.Lock()
	defer a.sessMu.Unlock()
	a.pruneSessionsLocked()
	s, ok := a.sessions[cookie.Value]
	if !ok {
		return ""
	}
	return s.Username
}

func (a *app) removeSession(r *http.Request) {
	cookie, err := r.Cookie(a.cfg.SessionCookieName)
	if err != nil || cookie == nil || cookie.Value == "" {
		return
	}
	a.sessMu.Lock()
	delete(a.sessions, cookie.Value)
	a.saveSessionsLocked()
	a.sessMu.Unlock()
}

func (a *app) updateSessionUsername(oldUsername, newUsername string) {
	a.sessMu.Lock()
	defer a.sessMu.Unlock()
	a.pruneSessionsLocked()
	changed := false
	for token, session := range a.sessions {
		if session.Username != oldUsername {
			continue
		}
		session.Username = newUsername
		a.sessions[token] = session
		changed = true
	}
	if changed {
		a.saveSessionsLocked()
	}
}

func (a *app) revokeOtherUserSessions(r *http.Request, username string) {
	currentToken := ""
	if cookie, err := r.Cookie(a.cfg.SessionCookieName); err == nil && cookie != nil {
		currentToken = cookie.Value
	}
	a.sessMu.Lock()
	defer a.sessMu.Unlock()
	changed := false
	for token, session := range a.sessions {
		if session.Username == username && token != currentToken {
			delete(a.sessions, token)
			changed = true
		}
	}
	if changed {
		a.saveSessionsLocked()
	}
}

func requestUsesHTTPS(r *http.Request) bool {
	if r != nil && r.TLS != nil {
		return true
	}
	return r != nil && strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
}

func (a *app) setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	expires := time.Now().Add(time.Duration(a.cfg.SessionTTLSeconds) * time.Second)
	http.SetCookie(w, &http.Cookie{
		Name:     a.cfg.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   requestUsesHTTPS(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   a.cfg.SessionTTLSeconds,
		Expires:  expires,
	})
}

func (a *app) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.cfg.SessionCookieName,
		Value:    "deleted",
		Path:     "/",
		HttpOnly: true,
		Secure:   requestUsesHTTPS(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	raw, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func writeBytes(w http.ResponseWriter, status int, body []byte, contentType string) {
	w.Header().Set("Content-Type", contentType)
	if w.Header().Get("Cache-Control") == "" {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func redirectNoStore(w http.ResponseWriter, location string, status int) {
	w.Header().Set("Location", location)
	w.Header().Set("Content-Length", "0")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
}

func readJSONBody(r *http.Request) (map[string]any, bool, bool) {
	if r.Body == nil {
		return map[string]any{}, true, false
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		return nil, false, false
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return map[string]any{}, true, false
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, false, true
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, true, false
}

func (a *app) requireAuth(w http.ResponseWriter, r *http.Request, requirePasswordChanged bool) (string, bool) {
	username := a.resolveSessionUsername(r)
	usersPayload := a.getUsersPayload()
	if username == "" || usersMap(usersPayload)[username] == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "Authentication required."})
		return "", false
	}
	if requirePasswordChanged && userRequiresPasswordChange(usersPayload, username) {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"message": "First-time setup required: change the account password before editing the startpage.",
		})
		return "", false
	}
	return username, true
}
