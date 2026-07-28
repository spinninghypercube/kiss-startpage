package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIconSourceRegistry(t *testing.T) {
	t.Parallel()

	items := iconSources()
	if len(items) != 15 {
		t.Fatalf("expected 15 icon sources, got %d", len(items))
	}

	seenIDs := map[string]bool{}
	seenPrefixes := map[string]bool{}
	for _, item := range items {
		if item.ID == "" || item.Label == "" || item.Provider == "" || item.LicenseName == "" {
			t.Fatalf("incomplete icon source: %#v", item)
		}
		if seenIDs[item.ID] {
			t.Fatalf("duplicate source ID %q", item.ID)
		}
		seenIDs[item.ID] = true
		if item.Provider == "iconify" {
			if item.Prefix == "" {
				t.Fatalf("missing prefix for %q", item.ID)
			}
			if seenPrefixes[item.Prefix] {
				t.Fatalf("duplicate Iconify prefix %q", item.Prefix)
			}
			seenPrefixes[item.Prefix] = true
		}
	}
	if len(iconProviders()) != 6 {
		t.Fatalf("expected 6 icon providers, got %d", len(iconProviders()))
	}
	if len(iconifyPrefixes()) != 14 {
		t.Fatalf("expected 14 Iconify prefixes, got %d", len(iconifyPrefixes()))
	}
}

func TestNormalizeIconifyNameUsesRegistry(t *testing.T) {
	t.Parallel()

	value, prefix, name, sourceID, err := normalizeIconifyName("home", "iconify-lucide")
	if err != nil {
		t.Fatalf("normalizeIconifyName returned error: %v", err)
	}
	if value != "lucide:home" || prefix != "lucide" || name != "home" || sourceID != "iconify-lucide" {
		t.Fatalf("unexpected normalized values: %q %q %q %q", value, prefix, name, sourceID)
	}

	if _, _, _, _, err := normalizeIconifyName("mdi:home", "iconify-lucide"); err == nil {
		t.Fatal("expected a mismatched source to be rejected")
	}
	if _, _, _, _, err := normalizeIconifyName("unknown:home", ""); err == nil {
		t.Fatal("expected an unknown prefix to be rejected")
	}
}

func TestIconSourcesEndpointAndIconifySearch(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search":
			if got := r.URL.Query().Get("prefixes"); got != strings.Join(iconifyPrefixes(), ",") {
				t.Errorf("unexpected prefixes query: %q", got)
			}
			writeJSON(w, http.StatusOK, map[string]any{"icons": []string{"lucide:home", "lucide:house"}})
		case r.URL.Path == "/lucide/home.svg":
			w.Header().Set("Content-Type", "image/svg+xml")
			_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	a, token := newAuthenticatedTestApp(t, upstream.URL)

	sourcesRequest := httptest.NewRequest(http.MethodGet, "/api/icons/sources", nil)
	sourcesResponse := httptest.NewRecorder()
	a.ServeHTTP(sourcesResponse, sourcesRequest)
	if sourcesResponse.Code != http.StatusOK {
		t.Fatalf("sources endpoint returned %d: %s", sourcesResponse.Code, sourcesResponse.Body.String())
	}
	var sourcesPayload struct {
		Items     []iconSourceDefinition   `json:"items"`
		Providers []iconProviderDefinition `json:"providers"`
	}
	if err := json.Unmarshal(sourcesResponse.Body.Bytes(), &sourcesPayload); err != nil {
		t.Fatalf("decode sources response: %v", err)
	}
	if len(sourcesPayload.Items) != 15 || len(sourcesPayload.Providers) != 6 {
		t.Fatalf("unexpected catalogue: %d sources, %d providers", len(sourcesPayload.Items), len(sourcesPayload.Providers))
	}

	if err := a.saveUserIconProviders("tester", []string{"iconify"}); err != nil {
		t.Fatal(err)
	}
	searchRequest := httptest.NewRequest(http.MethodGet, "/api/icons/search?q=home&limit=18", nil)
	searchRequest.AddCookie(&http.Cookie{Name: sessionCookieNameDefault, Value: token})
	searchResponse := httptest.NewRecorder()
	a.ServeHTTP(searchResponse, searchRequest)
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("search endpoint returned %d: %s", searchResponse.Code, searchResponse.Body.String())
	}
	var searchPayload struct {
		Groups []iconSearchGroup `json:"groups"`
	}
	if err := json.Unmarshal(searchResponse.Body.Bytes(), &searchPayload); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if len(searchPayload.Groups) != 1 || searchPayload.Groups[0].Provider != "iconify" || len(searchPayload.Groups[0].Items) != 2 {
		t.Fatalf("unexpected search payload: %#v", searchPayload)
	}
	if searchPayload.Groups[0].Items[0].Source != "iconify-lucide" || searchPayload.Groups[0].Items[0].Reference != "lucide:home" {
		t.Fatalf("unexpected search item: %#v", searchPayload.Groups[0].Items[0])
	}

	imported, status, err := a.fetchIconifyIconData("lucide:home", "svg", "iconify-lucide")
	if err != nil || status != http.StatusOK {
		t.Fatalf("fetchIconifyIconData failed: status=%d err=%v", status, err)
	}
	if imported["source"] != "iconify-lucide" || !strings.HasPrefix(imported["iconData"].(string), "data:image/svg+xml;base64,") {
		t.Fatalf("unexpected imported icon: %#v", imported)
	}
}

func TestIconPreferencesPersistAndFilterUnknownProviders(t *testing.T) {
	t.Parallel()

	a, token := newAuthenticatedTestApp(t, "http://127.0.0.1:1")
	request := httptest.NewRequest(http.MethodPost, "/api/icons/preferences", bytes.NewBufferString(`{"enabledProviders":["local","not-a-source"]}`))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: sessionCookieNameDefault, Value: token})
	response := httptest.NewRecorder()
	a.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if got := a.getUserIconProviders("tester"); len(got) != 1 || got[0] != "local" {
		t.Fatalf("unexpected stored providers: %#v", got)
	}
	payload := a.getUsersPayload()
	record := usersMap(payload)["tester"].(map[string]any)
	if record["marker"] != "preserve-me" {
		t.Fatalf("preference write replaced user record: %#v", record)
	}
}

func newAuthenticatedTestApp(t *testing.T, iconifyBase string) (*app, string) {
	t.Helper()

	dataDir := t.TempDir()
	usersFile := filepath.Join(dataDir, "users.json")
	if err := os.WriteFile(usersFile, []byte(`{"users":{"tester":{"marker":"preserve-me"}}}`), 0o600); err != nil {
		t.Fatalf("write users file: %v", err)
	}

	token := "test-session"
	return &app{
		cfg: envConfig{
			DataDir:            dataDir,
			PrivateIconsDir:    filepath.Join(dataDir, "private-icons"),
			ConfigFile:         filepath.Join(dataDir, "dashboard-config.json"),
			UsersFile:          usersFile,
			SessionsFile:       filepath.Join(dataDir, "sessions.json"),
			SessionTTLSeconds:  3600,
			SessionCookieName:  sessionCookieNameDefault,
			IconSearchMaxLimit: 30,
			IconifyAPIBase:     iconifyBase,
			SelfhstIndexURL:    iconifyBase + "/selfhst-index",
			SelfhstRawBase:     iconifyBase,
			DashboardIndexURL:  iconifyBase + "/dashboard-index",
			DashboardRawBase:   iconifyBase,
			WikimediaAPIBase:   iconifyBase + "/wikimedia",
		},
		client: &http.Client{Timeout: 2 * time.Second},
		sessions: map[string]sessionInfo{
			token: {Username: "tester", Expires: time.Now().Add(time.Hour).Unix()},
		},
	}, token
}
