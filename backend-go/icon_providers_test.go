package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestIconSearchReturnsPartialProviderResults(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer upstream.Close()

	a, token := newAuthenticatedTestApp(t, upstream.URL)
	if err := os.MkdirAll(a.cfg.PrivateIconsDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a.cfg.PrivateIconsDir, "home-local.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0h1v1z"/></svg>`), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := a.saveUserIconProviders("tester", []string{"iconify", "local"}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/icons/search?q=home&limit=10", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieNameDefault, Value: token})
	response := httptest.NewRecorder()
	a.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected partial 200 response, got %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Groups []iconSearchGroup `json:"groups"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Groups) != 2 || payload.Groups[0].Status != "error" || payload.Groups[1].Status != "ready" {
		t.Fatalf("unexpected provider statuses: %#v", payload.Groups)
	}
	if len(payload.Groups[1].Items) != 1 || payload.Groups[1].Items[0].Reference != "home-local.svg" {
		t.Fatalf("unexpected local results: %#v", payload.Groups[1].Items)
	}
}

func TestWikimediaStrictLicenseFilter(t *testing.T) {
	t.Parallel()

	base := map[string]wikimediaMetadataValue{
		"LicenseShortName":    {Value: "Public domain"},
		"AttributionRequired": {Value: "false"},
		"Copyrighted":         {Value: "False"},
		"Restrictions":        {Value: ""},
	}
	if !wikimediaItemAllowed(wikimediaImageInfo{ExtendedMetadata: base}) {
		t.Fatal("expected public-domain item to be allowed")
	}
	for key, value := range map[string]string{
		"LicenseShortName":    "CC BY 4.0",
		"AttributionRequired": "true",
		"Copyrighted":         "True",
		"Restrictions":        "trademarked",
	} {
		metadata := map[string]wikimediaMetadataValue{}
		for originalKey, originalValue := range base {
			metadata[originalKey] = originalValue
		}
		metadata[key] = wikimediaMetadataValue{Value: value}
		if wikimediaItemAllowed(wikimediaImageInfo{ExtendedMetadata: metadata}) {
			t.Fatalf("expected item with %s=%q to be rejected", key, value)
		}
	}
}

func TestWebsiteIconDiscoveryAndImport(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<html><head><link rel="icon" type="image/svg+xml" href="/site-icon.svg"></head></html>`))
		case "/site-icon.svg":
			w.Header().Set("Content-Type", "image/svg+xml")
			_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><circle cx="8" cy="8" r="8"/></svg>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	a, _ := newAuthenticatedTestApp(t, server.URL)
	items, err := a.searchWebsiteIcons([]string{server.URL})
	if err != nil || len(items) != 1 || !strings.HasSuffix(items[0].Reference, "/site-icon.svg") {
		t.Fatalf("unexpected website results: %#v err=%v", items, err)
	}
	imported, status, err := a.fetchWebsiteIconData(items[0].Reference)
	if err != nil || status != http.StatusOK {
		t.Fatalf("website import failed: status=%d err=%v", status, err)
	}
	data, _ := imported["iconData"].(string)
	if !strings.HasPrefix(data, "data:image/svg+xml;base64,") {
		t.Fatalf("unexpected website icon data: %q", data)
	}
	if _, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(data, "data:image/svg+xml;base64,")); err != nil {
		t.Fatalf("invalid base64 icon data: %v", err)
	}
}

func TestWebsitePageURLNormalization(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		raw           string
		defaultScheme string
		want          string
	}{
		"external bare domain":  {"example.com/path", "https", "https://example.com/path"},
		"internal bare address": {"internal.example.test:8080/app", "http", "http://internal.example.test:8080/app"},
		"protocol relative":     {"//example.com/icon", "https", "https://example.com/icon"},
		"qualified preserved":   {"http://example.com/path", "https", "http://example.com/path"},
		"unsupported scheme":    {"ftp://example.com/icon", "https", ""},
		"credentials rejected":  {"https://user:pass@example.com", "https", ""},
		"relative rejected":     {"/relative/path", "https", ""},
	} {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeWebsitePageURL(testCase.raw, testCase.defaultScheme); got != testCase.want {
				t.Fatalf("normalizeWebsitePageURL(%q, %q) = %q, want %q", testCase.raw, testCase.defaultScheme, got, testCase.want)
			}
		})
	}
}

func TestIconSearchOnlyReturnsRelevantProviderGroups(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<link rel="icon" href="/favicon.svg">`))
		case "/favicon.svg":
			w.Header().Set("Content-Type", "image/svg+xml")
			_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	a, token := newAuthenticatedTestApp(t, server.URL)
	if err := a.saveUserIconProviders("tester", []string{"iconify", "website"}); err != nil {
		t.Fatal(err)
	}

	search := func(target string) []iconSearchGroup {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, target, nil)
		request.AddCookie(&http.Cookie{Name: sessionCookieNameDefault, Value: token})
		response := httptest.NewRecorder()
		a.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("search failed with %d: %s", response.Code, response.Body.String())
		}
		var payload struct {
			Groups []iconSearchGroup `json:"groups"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload.Groups
	}

	if groups := search("/api/icons/search?q="); len(groups) != 0 {
		t.Fatalf("expected no groups before search input, got %#v", groups)
	}
	bareServerAddress := strings.TrimPrefix(server.URL, "http://")
	websiteTarget := "/api/icons/search?q=&internalUrl=" + url.QueryEscape(bareServerAddress)
	if groups := search(websiteTarget); len(groups) != 1 || groups[0].Provider != "website" || groups[0].Status != "ready" ||
		len(groups[0].Items) != 1 || groups[0].Items[0].SourceURL != server.URL {
		t.Fatalf("expected only a ready website group, got %#v", groups)
	}
	externalTarget := "/api/icons/search?q=&externalUrl=" + url.QueryEscape(bareServerAddress)
	if groups := search(externalTarget); len(groups) != 1 || len(groups[0].Items) != 1 ||
		!strings.HasPrefix(groups[0].Items[0].SourceURL, "https://") {
		t.Fatalf("expected scheme-less external URL to default to HTTPS, got %#v", groups)
	}
}

func TestUnsafeSVGIsRejected(t *testing.T) {
	t.Parallel()

	if _, err := safeImageContent([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), "image/svg+xml", "bad.svg"); err == nil {
		t.Fatal("expected scripted SVG to be rejected")
	}
	if contentType, err := safeImageContent([]byte(`<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0h1"/></svg>`), "image/svg+xml", "safe.svg"); err != nil || contentType != "image/svg+xml" {
		t.Fatalf("expected safe SVG, got contentType=%q err=%v", contentType, err)
	}
}

func TestLimitedFetchRetriesTransientUpstreamErrors(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 3 {
			http.Error(w, "temporary failure", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("\x89PNG\r\n\x1a\n"))
	}))
	defer server.Close()

	a, _ := newAuthenticatedTestApp(t, server.URL)
	raw, contentType, status, err := a.fetchLimitedWithRetry(server.URL, "image/png", 1024, 2*time.Second, 3)
	if err != nil || status != http.StatusOK || contentType != "image/png" || len(raw) == 0 {
		t.Fatalf("expected retry to recover, status=%d contentType=%q err=%v", status, contentType, err)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts.Load())
	}
}
