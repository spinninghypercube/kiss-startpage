package main

import (
	"errors"
	"net/http"
	"os"
	"strings"
)

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch r.Method {
	case http.MethodGet:
		a.handleGET(w, r, path)
	case http.MethodPost:
		a.handlePOST(w, r, path)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Not found."})
	}
}

func (a *app) handleGET(w http.ResponseWriter, r *http.Request, path string) {
	switch path {
	case "/health":
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	case "/api/version":
		writeJSON(w, http.StatusOK, map[string]any{"version": appVersion})
		return
	case "/api/config":
		writeJSON(w, http.StatusOK, map[string]any{"config": a.getConfigPayload()})
		return
	case "/api/auth/status":
		username := a.resolveSessionUsername(r)
		if username != "" {
			writeJSON(w, http.StatusOK, map[string]any{
				"authenticated":      true,
				"username":           username,
				"mustChangePassword": a.isPasswordChangeRequired(username),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated": false,
			"setupRequired": !hasAnyUsers(a.getUsersPayload()),
		})
		return
	case "/api/icons/sources":
		writeJSON(w, http.StatusOK, map[string]any{"items": iconSources(), "providers": iconProviders()})
		return
	case "/api/icons/preferences":
		username, ok := a.requireAuth(w, r, true)
		if !ok {
			return
		}
		a.handleGetIconPreferences(w, username)
		return
	case "/api/icons/search":
		username, ok := a.requireAuth(w, r, true)
		if !ok {
			return
		}
		a.handleAllIconSearch(w, r, username)
		return
	default:
		if a.serveStatic(w, r) {
			return
		}
		if a.servePrivateIcon(w, r, path) {
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Not found."})
		return
	}
}

func (a *app) handlePOST(w http.ResponseWriter, r *http.Request, path string) {
	payload, ok, invalid := readJSONBody(r)
	if !ok {
		if invalid {
			writeJSON(w, http.StatusBadRequest, map[string]any{"message": "Invalid JSON body."})
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]any{"message": "Failed to read request body."})
		}
		return
	}
	switch path {
	case "/api/auth/bootstrap":
		a.handleBootstrap(w, r, payload)
	case "/api/login":
		a.handleLogin(w, r, payload)
	case "/api/logout":
		a.removeSession(r)
		a.clearSessionCookie(w, r)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "/api/config":
		a.handleSaveConfig(w, r, payload)
	case "/api/auth/change-username":
		a.handleChangeUsername(w, r, payload)
	case "/api/auth/change-password":
		a.handleChangePassword(w, r, payload)
	case "/api/icons/preferences":
		a.handleSaveIconPreferences(w, r, payload)
	case "/api/icons/import":
		a.handleImportIcon(w, r, payload)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Not found."})
	}
}

func (a *app) handleBootstrap(w http.ResponseWriter, r *http.Request, payload map[string]any) {
	if hasAnyUsers(a.getUsersPayload()) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"message":       "An admin account is already configured.",
			"setupRequired": false,
		})
		return
	}
	username := strings.TrimSpace(asString(payload["username"]))
	password := asString(payload["password"])
	if !validUsername(username) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"message": "Username must be 3-40 chars and use only letters, numbers, dot, dash or underscore.",
		})
		return
	}
	if len(password) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"message": "Password must be at least 4 characters."})
		return
	}
	rec, err := buildPasswordRecord(password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to create account."})
		return
	}
	err = a.updateUsersPayload(func(usersPayload map[string]any) error {
		users := usersMap(usersPayload)
		if len(users) > 0 {
			return os.ErrExist
		}
		users[username] = rec
		return nil
	})
	if errors.Is(err, os.ErrExist) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"message":       "An admin account is already configured.",
			"setupRequired": false,
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to save users."})
		return
	}
	token, err := a.createSession(username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to create session."})
		return
	}
	a.setSessionCookie(w, r, token)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                 true,
		"username":           username,
		"mustChangePassword": false,
		"setupRequired":      false,
	})
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request, payload map[string]any) {
	username := strings.TrimSpace(asString(payload["username"]))
	password := asString(payload["password"])
	if username == "" || password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"message": "Username and password are required."})
		return
	}
	usersPayload := a.getUsersPayload()
	if !hasAnyUsers(usersPayload) {
		writeJSON(w, http.StatusConflict, map[string]any{
			"message":       "No admin account configured yet. Complete first-time setup.",
			"setupRequired": true,
		})
		return
	}
	rec := usersMap(usersPayload)[username]
	if !verifyPassword(password, rec) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "Invalid username or password."})
		return
	}
	token, err := a.createSession(username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to create session."})
		return
	}
	a.setSessionCookie(w, r, token)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                 true,
		"username":           username,
		"mustChangePassword": userRequiresPasswordChange(usersPayload, username),
	})
}

func (a *app) handleSaveConfig(w http.ResponseWriter, r *http.Request, payload map[string]any) {
	username, ok := a.requireAuth(w, r, true)
	if !ok {
		return
	}
	cfg, hasCfg := payload["config"]
	if !hasCfg {
		writeJSON(w, http.StatusBadRequest, map[string]any{"message": "Missing config payload."})
		return
	}
	saved, err := a.saveConfigPayload(cfg)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to save config."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"savedBy": username,
		"config":  saved,
	})
}

func (a *app) handleChangeUsername(w http.ResponseWriter, r *http.Request, payload map[string]any) {
	username, ok := a.requireAuth(w, r, false)
	if !ok {
		return
	}
	currentPassword := asString(payload["currentPassword"])
	newUsername := strings.TrimSpace(asString(payload["newUsername"]))
	if !validUsername(newUsername) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"message": "Username must be 3-40 chars and use only letters, numbers, dot, dash or underscore.",
		})
		return
	}
	var passwordIncorrect bool
	var usernameExists bool
	err := a.updateUsersPayload(func(usersPayload map[string]any) error {
		users := usersMap(usersPayload)
		rec := users[username]
		if !verifyPassword(currentPassword, rec) {
			passwordIncorrect = true
			return nil
		}
		if newUsername == username {
			return nil
		}
		if _, exists := users[newUsername]; exists {
			usernameExists = true
			return nil
		}
		users[newUsername] = users[username]
		delete(users, username)
		return nil
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to save users."})
		return
	}
	if passwordIncorrect {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "Current password is incorrect."})
		return
	}
	if usernameExists {
		writeJSON(w, http.StatusConflict, map[string]any{"message": "Username already exists."})
		return
	}
	if newUsername != username {
		a.updateSessionUsername(username, newUsername)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "username": newUsername})
}

func (a *app) handleChangePassword(w http.ResponseWriter, r *http.Request, payload map[string]any) {
	username, ok := a.requireAuth(w, r, false)
	if !ok {
		return
	}
	currentPassword := asString(payload["currentPassword"])
	newPassword := asString(payload["newPassword"])
	if len(newPassword) < 4 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"message": "New password must be at least 4 characters."})
		return
	}
	newRec, err := buildPasswordRecord(newPassword)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to update password."})
		return
	}
	var passwordIncorrect bool
	err = a.updateUsersPayload(func(usersPayload map[string]any) error {
		users := usersMap(usersPayload)
		rec := users[username]
		if !verifyPassword(currentPassword, rec) {
			passwordIncorrect = true
			return nil
		}
		if oldRec, ok := rec.(map[string]any); ok {
			if preferences, exists := oldRec["preferences"]; exists {
				newRec["preferences"] = preferences
			}
		}
		users[username] = newRec
		clearPasswordChangeRequired(usersPayload, username)
		return nil
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to save users."})
		return
	}
	if passwordIncorrect {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"message": "Current password is incorrect."})
		return
	}
	a.revokeOtherUserSessions(r, username)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "mustChangePassword": false})
}
