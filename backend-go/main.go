package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	appVersion               = "2.8.0"
	sessionCookieNameDefault = "dash_session"
	defaultSessionTTL        = 315360000
	defaultIconIndexTTL      = 21600
	defaultIconSearchLimit   = 30
)

var (
	validUsernameRe  = regexp.MustCompile(`^[A-Za-z0-9._-]{3,40}$`)
	selfhstRefRe     = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
	iconifyNameRe    = regexp.MustCompile(`^[a-z0-9-]+:[a-z0-9][a-z0-9._-]*$`)
	truthyFlagValues = map[string]bool{"y": true, "yes": true, "true": true, "1": true}
)

type envConfig struct {
	Bind               string
	Port               int
	DataDir            string
	PrivateIconsDir    string
	ConfigFile         string
	UsersFile          string
	DefaultConfigPath  string
	AppRoot            string
	SessionTTLSeconds  int
	SessionCookieName  string
	SessionsFile       string
	IconIndexTTL       int
	IconSearchMaxLimit int
	SelfhstIndexURL    string
	SelfhstRawBase     string
	IconifyAPIBase     string
	DashboardIndexURL  string
	DashboardRawBase   string
	WikimediaAPIBase   string
}

type sessionInfo struct {
	Username string
	Expires  int64
}

type selfhstIcon struct {
	Name      string `json:"name"`
	Reference string `json:"reference"`
	Category  string `json:"category"`
	Tags      string `json:"tags"`
	HasSVG    bool   `json:"hasSvg"`
	HasPNG    bool   `json:"hasPng"`
	HasWebP   bool   `json:"hasWebp"`
	HasLight  bool   `json:"hasLight"`
	HasDark   bool   `json:"hasDark"`
}

type iconSearchResult struct {
	Score      int    `json:"score,omitempty"`
	Name       string `json:"name"`
	Reference  string `json:"reference"`
	Category   string `json:"category"`
	Tags       string `json:"tags"`
	HasSVG     bool   `json:"hasSvg,omitempty"`
	HasPNG     bool   `json:"hasPng,omitempty"`
	HasWebP    bool   `json:"hasWebp,omitempty"`
	HasLight   bool   `json:"hasLight,omitempty"`
	HasDark    bool   `json:"hasDark,omitempty"`
	PreviewURL string `json:"previewUrl"`
	Source     string `json:"source,omitempty"`
	Provider   string `json:"provider,omitempty"`
	License    string `json:"license,omitempty"`
	LicenseURL string `json:"licenseUrl,omitempty"`
	SourceURL  string `json:"sourceUrl,omitempty"`
}

type iconCache struct {
	FetchedAt int64
	Items     []selfhstIcon
}

type app struct {
	cfg      envConfig
	client   *http.Client
	fileMu   sync.Mutex
	sessMu   sync.Mutex
	sessions map[string]sessionInfo
	iconMu   sync.Mutex
	iconIdx  iconCache
	dashIdx  iconCache
}

func main() {
	cfg := loadEnv()
	a := &app{
		cfg:      cfg,
		client:   &http.Client{Timeout: 20 * time.Second},
		sessions: map[string]sessionInfo{},
	}
	if err := a.ensureFilesReady(); err != nil {
		log.Fatalf("failed to initialize data files: %v", err)
	}
	a.loadSessions()

	addr := fmt.Sprintf("%s:%d", cfg.Bind, cfg.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           a,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("KISS Startpage v%s — listening on http://%s", appVersion, addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

func loadEnv() envConfig {
	exe, _ := os.Executable()
	defaultAppRoot := filepath.Dir(filepath.Dir(exe))
	if defaultAppRoot == "." || defaultAppRoot == "/" {
		defaultAppRoot = cwdOrFallback()
	}
	dataDir := getenv("DASH_DATA_DIR", "/srv/www/kiss-startpage/shared/data")
	privateIconsDir := getenv("DASH_PRIVATE_ICONS_DIR", filepath.Join(filepath.Dir(dataDir), "private-icons"))
	appRoot := getenv("DASH_APP_ROOT", defaultAppRoot)
	return envConfig{
		Bind:               getenv("DASH_BIND", "127.0.0.1"),
		Port:               getenvInt("DASH_PORT", 8788),
		DataDir:            dataDir,
		PrivateIconsDir:    privateIconsDir,
		ConfigFile:         filepath.Join(dataDir, "dashboard-config.json"),
		UsersFile:          filepath.Join(dataDir, "users.json"),
		DefaultConfigPath:  getenv("DASH_DEFAULT_CONFIG", "/srv/www/kiss-startpage/current/startpage-default-config.json"),
		AppRoot:            appRoot,
		SessionTTLSeconds:  getenvInt("DASH_SESSION_TTL", defaultSessionTTL),
		SessionCookieName:  sessionCookieNameDefault,
		SessionsFile:       filepath.Join(dataDir, "sessions.json"),
		IconIndexTTL:       getenvInt("DASH_ICON_INDEX_TTL", defaultIconIndexTTL),
		IconSearchMaxLimit: getenvInt("DASH_ICON_SEARCH_MAX_LIMIT", defaultIconSearchLimit),
		SelfhstIndexURL:    getenv("DASH_ICON_INDEX_URL", "https://raw.githubusercontent.com/selfhst/icons/main/index.json"),
		SelfhstRawBase:     getenv("DASH_ICON_RAW_BASE", "https://raw.githubusercontent.com/selfhst/icons/main"),
		IconifyAPIBase:     getenv("DASH_ICONIFY_API_BASE", "https://api.iconify.design"),
		DashboardIndexURL:  getenv("DASH_DASHBOARD_ICON_INDEX_URL", "https://raw.githubusercontent.com/homarr-labs/dashboard-icons/main/metadata.json"),
		DashboardRawBase:   getenv("DASH_DASHBOARD_ICON_RAW_BASE", "https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons"),
		WikimediaAPIBase:   getenv("DASH_WIKIMEDIA_API_BASE", "https://commons.wikimedia.org/w/api.php"),
	}
}

func cwdOrFallback() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
