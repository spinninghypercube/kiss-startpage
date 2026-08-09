package main

import (
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func (a *app) serveStatic(w http.ResponseWriter, r *http.Request) bool {
	rawPath := r.URL.Path
	querySuffix := ""
	if r.URL.RawQuery != "" {
		querySuffix = "?" + r.URL.RawQuery
	}
	switch rawPath {
	case "/index.html":
		redirectNoStore(w, "/"+querySuffix, http.StatusFound)
		return true
	case "/admin.html", "/admin", "/edit.html":
		redirectNoStore(w, "/edit"+querySuffix, http.StatusFound)
		return true
	case "/edit/":
		redirectNoStore(w, "/edit"+querySuffix, http.StatusFound)
		return true
	}

	relative := ""
	switch {
	case rawPath == "/edit":
		relative = "index.html"
	case rawPath == "/" || rawPath == "":
		relative = "index.html"
	default:
		relative = strings.TrimPrefix(rawPath, "/")
		if strings.HasSuffix(rawPath, "/") {
			relative = strings.TrimSuffix(relative, "/") + "/index.html"
		}
	}
	relative = filepath.ToSlash(filepath.Clean(relative))
	relative = strings.TrimPrefix(relative, "/")
	if relative == "." || relative == "" {
		relative = "index.html"
	}
	if strings.HasPrefix(relative, "backend/") ||
		strings.HasPrefix(relative, "ops/") ||
		strings.HasPrefix(relative, ".") ||
		strings.Contains(relative, "/.") ||
		relative == ".." ||
		strings.HasPrefix(relative, "../") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(relative))
	allowed := map[string]bool{
		".html": true, ".js": true, ".json": true, ".css": true, ".svg": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
		".ico": true, ".txt": true, ".webmanifest": true,
	}
	if !allowed[ext] {
		return false
	}
	appRoot, err := filepath.Abs(a.cfg.AppRoot)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to resolve app root."})
		return true
	}
	filePath, err := filepath.Abs(filepath.Join(appRoot, relative))
	if err != nil {
		return false
	}
	if filePath != appRoot && !strings.HasPrefix(filePath, appRoot+string(os.PathSeparator)) {
		return false
	}
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return false
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to read static file."})
		return true
	}
	contentType := mime.TypeByExtension(ext)
	switch ext {
	case ".js":
		contentType = "application/javascript; charset=utf-8"
	case ".html":
		contentType = "text/html; charset=utf-8"
	case ".json":
		contentType = "application/json; charset=utf-8"
	case ".webmanifest":
		contentType = "application/manifest+json; charset=utf-8"
	case ".css":
		contentType = "text/css; charset=utf-8"
	default:
		if strings.HasPrefix(contentType, "text/") {
			contentType += "; charset=utf-8"
		}
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if ext == ".html" || ext == ".webmanifest" {
		w.Header().Set("Cache-Control", "no-cache")
	} else if strings.HasPrefix(relative, "assets/index-") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
	writeBytes(w, http.StatusOK, raw, contentType)
	return true
}

func (a *app) servePrivateIcon(w http.ResponseWriter, r *http.Request, rawPath string) bool {
	if !strings.HasPrefix(rawPath, "/icons/") {
		return false
	}
	name, err := url.PathUnescape(strings.TrimPrefix(rawPath, "/icons/"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"message": "Invalid icon path."})
		return true
	}
	name = strings.TrimSpace(name)
	if name == "" ||
		strings.Contains(name, "/") ||
		strings.Contains(name, "\\") ||
		strings.Contains(name, "..") ||
		strings.HasPrefix(name, ".") {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Not found."})
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	allowed := map[string]bool{
		".svg": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".ico": true,
	}
	if !allowed[ext] {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Not found."})
		return true
	}
	iconRoot, err := filepath.Abs(a.cfg.PrivateIconsDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to resolve icon path."})
		return true
	}
	filePath, err := filepath.Abs(filepath.Join(iconRoot, name))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Not found."})
		return true
	}
	if filePath != iconRoot && !strings.HasPrefix(filePath, iconRoot+string(os.PathSeparator)) {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Not found."})
		return true
	}
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		writeJSON(w, http.StatusNotFound, map[string]any{"message": "Not found."})
		return true
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to read icon file."})
		return true
	}
	contentType := mime.TypeByExtension(ext)
	if ext == ".svg" {
		contentType = "image/svg+xml"
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	writeBytes(w, http.StatusOK, raw, contentType)
	return true
}
