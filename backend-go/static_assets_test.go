package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServeStaticWebManifest(t *testing.T) {
	appRoot := t.TempDir()
	manifest := []byte(`{"name":"KISS Startpage"}`)
	if err := os.WriteFile(filepath.Join(appRoot, "manifest.webmanifest"), manifest, 0o644); err != nil {
		t.Fatalf("write manifest fixture: %v", err)
	}

	a := &app{cfg: envConfig{AppRoot: appRoot}}
	request := httptest.NewRequest(http.MethodGet, "/manifest.webmanifest", nil)
	response := httptest.NewRecorder()
	a.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/manifest+json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := response.Body.Bytes(); string(got) != string(manifest) {
		t.Fatalf("body = %q, want %q", got, manifest)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
}

func TestServeStaticHashedAssetUsesImmutableCache(t *testing.T) {
	appRoot := t.TempDir()
	assetDir := filepath.Join(appRoot, "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "index-ab12cd34.js"), []byte("export {};"), 0o644); err != nil {
		t.Fatal(err)
	}

	a := &app{cfg: envConfig{AppRoot: appRoot}}
	response := httptest.NewRecorder()
	a.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/assets/index-ab12cd34.js", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", got)
	}
}
