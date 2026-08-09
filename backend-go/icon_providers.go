package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	maxWebsitePageBytes  = 1024 * 1024
	maxImportedIconBytes = 2 * 1024 * 1024
)

type iconSearchGroup struct {
	Provider string             `json:"provider"`
	Label    string             `json:"label"`
	Status   string             `json:"status"`
	Message  string             `json:"message,omitempty"`
	Items    []iconSearchResult `json:"items"`
}

type dashboardIconMetadata struct {
	Base       string            `json:"base"`
	Aliases    []string          `json:"aliases"`
	Categories []string          `json:"categories"`
	Colors     map[string]string `json:"colors"`
}

type wikimediaMetadataValue struct {
	Value any `json:"value"`
}

type wikimediaImageInfo struct {
	URL              string                            `json:"url"`
	ThumbURL         string                            `json:"thumburl"`
	DescriptionURL   string                            `json:"descriptionurl"`
	MIME             string                            `json:"mime"`
	ExtendedMetadata map[string]wikimediaMetadataValue `json:"extmetadata"`
}

type wikimediaPage struct {
	PageID    int                  `json:"pageid"`
	Title     string               `json:"title"`
	ImageInfo []wikimediaImageInfo `json:"imageinfo"`
}

type wikimediaQueryResponse struct {
	Query struct {
		Pages []wikimediaPage `json:"pages"`
	} `json:"query"`
}

var (
	faviconLinkRe = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	htmlAttrRe    = regexp.MustCompile(`(?i)([a-z_:][-a-z0-9_:.]*)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>` + "`" + `]+))`)
)

func normalizeProviderIDs(raw any) []string {
	values := []string{}
	switch typed := raw.(type) {
	case []any:
		for _, value := range typed {
			values = append(values, asString(value))
		}
	case []string:
		values = append(values, typed...)
	}
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		id := strings.ToLower(strings.TrimSpace(value))
		if seen[id] {
			continue
		}
		if _, ok := iconProviderByID(id); !ok {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}

func (a *app) getUserIconProviders(username string) []string {
	payload := a.getUsersPayload()
	record, _ := usersMap(payload)[username].(map[string]any)
	preferences, _ := record["preferences"].(map[string]any)
	raw, exists := preferences["enabledIconProviders"]
	if !exists {
		return defaultIconProviderIDs()
	}
	return normalizeProviderIDs(raw)
}

func (a *app) saveUserIconProviders(username string, providers []string) error {
	return a.updateUsersPayload(func(payload map[string]any) error {
		users := usersMap(payload)
		record, _ := users[username].(map[string]any)
		if record == nil {
			return fmt.Errorf("user not found")
		}
		preferences, _ := record["preferences"].(map[string]any)
		if preferences == nil {
			preferences = map[string]any{}
		}
		stored := make([]any, 0, len(providers))
		for _, provider := range providers {
			stored = append(stored, provider)
		}
		preferences["enabledIconProviders"] = stored
		record["preferences"] = preferences
		users[username] = record
		return nil
	})
}

func (a *app) handleGetIconPreferences(w http.ResponseWriter, username string) {
	writeJSON(w, http.StatusOK, map[string]any{
		"enabledProviders": a.getUserIconProviders(username),
		"providers":        iconProviders(),
	})
}

func (a *app) handleSaveIconPreferences(w http.ResponseWriter, r *http.Request, payload map[string]any) {
	username, ok := a.requireAuth(w, r, true)
	if !ok {
		return
	}
	raw, exists := payload["enabledProviders"]
	if !exists {
		writeJSON(w, http.StatusBadRequest, map[string]any{"message": "Missing enabledProviders."})
		return
	}
	providers := normalizeProviderIDs(raw)
	if err := a.saveUserIconProviders(username, providers); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"message": "Failed to save icon preferences."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "enabledProviders": providers})
}

func (a *app) handleAllIconSearch(w http.ResponseWriter, r *http.Request, username string) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := asInt(r.URL.Query().Get("limit"), 18)
	if limit < 1 {
		limit = 1
	}
	if limit > a.cfg.IconSearchMaxLimit {
		limit = a.cfg.IconSearchMaxLimit
	}
	contextURLs := []string{}
	if externalURL := normalizeWebsitePageURL(r.URL.Query().Get("externalUrl"), "https"); externalURL != "" {
		contextURLs = append(contextURLs, externalURL)
	}
	if internalURL := normalizeWebsitePageURL(r.URL.Query().Get("internalUrl"), "http"); internalURL != "" {
		contextURLs = append(contextURLs, internalURL)
	}
	enabled := a.getUserIconProviders(username)
	searchableProviders := make([]string, 0, len(enabled))
	for _, providerID := range enabled {
		if providerID == "website" {
			if len(contextURLs) > 0 {
				searchableProviders = append(searchableProviders, providerID)
			}
			continue
		}
		if len(query) >= 2 {
			searchableProviders = append(searchableProviders, providerID)
		}
	}
	groups := make([]iconSearchGroup, len(searchableProviders))
	var wg sync.WaitGroup
	for index, providerID := range searchableProviders {
		index, providerID := index, providerID
		provider, ok := iconProviderByID(providerID)
		if !ok {
			continue
		}
		groups[index] = iconSearchGroup{
			Provider: provider.ID,
			Label:    provider.Label,
			Status:   "loading",
			Items:    []iconSearchResult{},
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			items, status, message, err := a.searchIconProvider(providerID, query, limit, contextURLs)
			groups[index].Items = items
			groups[index].Status = status
			groups[index].Message = message
			if err != nil {
				groups[index].Status = "error"
				groups[index].Message = err.Error()
			}
		}()
	}
	wg.Wait()
	writeJSON(w, http.StatusOK, map[string]any{
		"query":            query,
		"enabledProviders": enabled,
		"groups":           groups,
	})
}

func (a *app) searchIconProvider(providerID, query string, limit int, contextURLs []string) ([]iconSearchResult, string, string, error) {
	if providerID != "website" && len(strings.TrimSpace(query)) < 2 {
		return []iconSearchResult{}, "idle", "Enter at least 2 characters.", nil
	}
	var (
		items []iconSearchResult
		err   error
	)
	switch providerID {
	case "iconify":
		items, err = a.searchAllIconifyIcons(query, limit)
	case "selfhst":
		items, err = a.searchSelfhstIcons(query, limit)
		for index := range items {
			items[index].Provider = "selfhst"
			items[index].Source = "selfhst"
			items[index].License = "CC BY 4.0"
			items[index].LicenseURL = "https://github.com/selfhst/icons/blob/main/LICENSE"
		}
	case "dashboard":
		items, err = a.searchDashboardIcons(query, limit)
	case "local":
		items, err = a.searchLocalIcons(query, limit)
	case "website":
		items, err = a.searchWebsiteIcons(contextURLs)
	case "wikimedia":
		items, err = a.searchWikimediaIcons(query, limit)
	default:
		return []iconSearchResult{}, "error", "", fmt.Errorf("unsupported icon provider")
	}
	if err != nil {
		return []iconSearchResult{}, "error", "", err
	}
	if len(items) == 0 {
		if providerID == "website" && strings.TrimSpace(strings.Join(contextURLs, "")) == "" {
			return items, "idle", "Enter a button URL to discover its website icon.", nil
		}
		return items, "empty", "No icons found.", nil
	}
	return items, "ready", fmt.Sprintf("Found %d icon(s).", len(items)), nil
}

func (a *app) searchAllIconifyIcons(query string, limit int) ([]iconSearchResult, error) {
	prefixes := iconifyPrefixes()
	requestLimit := limit
	if requestLimit < 32 {
		requestLimit = 32
	}
	searchURL := fmt.Sprintf("%s/search?query=%s&limit=%d&prefixes=%s",
		strings.TrimRight(a.cfg.IconifyAPIBase, "/"),
		url.QueryEscape(strings.TrimSpace(query)),
		requestLimit,
		url.QueryEscape(strings.Join(prefixes, ",")),
	)
	resp, err := a.httpGet(searchURL, 20*time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Iconify search unavailable (%d)", resp.StatusCode)
	}
	var parsed struct {
		Icons []string `json:"icons"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	results := []iconSearchResult{}
	for _, iconName := range parsed.Icons {
		parts := strings.SplitN(iconName, ":", 2)
		if len(parts) != 2 || parts[1] == "" {
			continue
		}
		source, ok := iconSourceByPrefix(parts[0])
		if !ok {
			continue
		}
		results = append(results, iconSearchResult{
			Name:       titleFromSlug(parts[1]),
			Reference:  iconName,
			Category:   strings.TrimPrefix(source.Label, "Iconify · "),
			PreviewURL: fmt.Sprintf("%s/%s/%s.svg", strings.TrimRight(a.cfg.IconifyAPIBase, "/"), url.PathEscape(parts[0]), url.PathEscape(parts[1])),
			Source:     source.ID,
			Provider:   "iconify",
			License:    source.LicenseName,
			LicenseURL: source.LicenseURL,
		})
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func titleFromSlug(value string) string {
	words := strings.Fields(strings.NewReplacer("-", " ", "_", " ").Replace(value))
	for index, word := range words {
		if word != "" {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[index] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

func (a *app) getDashboardIconIndex() ([]selfhstIcon, error) {
	now := time.Now().Unix()
	a.iconMu.Lock()
	cache := a.dashIdx
	a.iconMu.Unlock()
	if len(cache.Items) > 0 && now-cache.FetchedAt < int64(a.cfg.IconIndexTTL) {
		return cache.Items, nil
	}
	resp, err := a.httpGet(a.cfg.DashboardIndexURL, 20*time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Dashboard Icons index unavailable (%d)", resp.StatusCode)
	}
	var metadata map[string]dashboardIconMetadata
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4*1024*1024)).Decode(&metadata); err != nil {
		return nil, err
	}
	items := make([]selfhstIcon, 0, len(metadata))
	for reference, item := range metadata {
		base := strings.ToLower(strings.TrimSpace(item.Base))
		if reference == "" || (base != "svg" && base != "png" && base != "webp") {
			continue
		}
		items = append(items, selfhstIcon{
			Name:      titleFromSlug(reference),
			Reference: reference,
			Category:  strings.Join(item.Categories, ", "),
			Tags:      strings.Join(item.Aliases, " "),
			HasSVG:    base == "svg",
			HasPNG:    base == "png",
			HasWebP:   base == "webp",
		})
	}
	a.iconMu.Lock()
	a.dashIdx = iconCache{FetchedAt: now, Items: items}
	a.iconMu.Unlock()
	return items, nil
}

func searchIndexedIcons(items []selfhstIcon, query string, limit int, result func(selfhstIcon) iconSearchResult) []iconSearchResult {
	normalized := strings.ToLower(strings.TrimSpace(query))
	tokens := strings.Fields(normalized)
	type scoredResult struct {
		score int
		item  selfhstIcon
	}
	scored := []scoredResult{}
	for _, item := range items {
		name := strings.ToLower(item.Name)
		reference := strings.ToLower(item.Reference)
		haystack := name + " " + reference + " " + strings.ToLower(item.Category) + " " + strings.ToLower(item.Tags)
		if !strings.Contains(haystack, normalized) {
			continue
		}
		score := 100
		if reference == normalized || name == normalized {
			score += 1000
		} else if strings.HasPrefix(reference, normalized) || strings.HasPrefix(name, normalized) {
			score += 700
		}
		for _, token := range tokens {
			if strings.Contains(reference, token) || strings.Contains(name, token) {
				score += 100
			}
		}
		scored = append(scored, scoredResult{score: score, item: item})
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return strings.ToLower(scored[i].item.Name) < strings.ToLower(scored[j].item.Name)
	})
	results := []iconSearchResult{}
	for _, entry := range scored {
		results = append(results, result(entry.item))
		if len(results) >= limit {
			break
		}
	}
	return results
}

func (a *app) searchDashboardIcons(query string, limit int) ([]iconSearchResult, error) {
	items, err := a.getDashboardIconIndex()
	if err != nil {
		return nil, err
	}
	return searchIndexedIcons(items, query, limit, func(item selfhstIcon) iconSearchResult {
		extension := "png"
		if item.HasSVG {
			extension = "svg"
		} else if item.HasWebP {
			extension = "webp"
		}
		return iconSearchResult{
			Name:       item.Name,
			Reference:  item.Reference,
			Category:   item.Category,
			Tags:       item.Tags,
			PreviewURL: fmt.Sprintf("%s/%s/%s.%s", strings.TrimRight(a.cfg.DashboardRawBase, "/"), extension, url.PathEscape(item.Reference), extension),
			Source:     "dashboard",
			Provider:   "dashboard",
			License:    "Apache-2.0",
			LicenseURL: "https://github.com/homarr-labs/dashboard-icons/blob/main/LICENSE",
		}
	}), nil
}

func allowedLocalIconExtension(extension string) bool {
	switch strings.ToLower(extension) {
	case ".svg", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico":
		return true
	default:
		return false
	}
}

func (a *app) searchLocalIcons(query string, limit int) ([]iconSearchResult, error) {
	entries, err := os.ReadDir(a.cfg.PrivateIconsDir)
	if errorsIsNotExist(err) {
		return []iconSearchResult{}, nil
	}
	if err != nil {
		return nil, err
	}
	normalized := strings.ToLower(strings.TrimSpace(query))
	items := []iconSearchResult{}
	for _, entry := range entries {
		if entry.IsDir() || !allowedLocalIconExtension(filepath.Ext(entry.Name())) {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if !strings.Contains(strings.ToLower(name+" "+entry.Name()), normalized) {
			continue
		}
		items = append(items, iconSearchResult{
			Name:       titleFromSlug(name),
			Reference:  entry.Name(),
			Category:   "Private server icon",
			PreviewURL: "/icons/" + url.PathEscape(entry.Name()),
			Source:     "local",
			Provider:   "local",
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}

func parseHTMLAttributes(tag string) map[string]string {
	attributes := map[string]string{}
	for _, match := range htmlAttrRe.FindAllStringSubmatch(tag, -1) {
		value := match[2]
		if value == "" {
			value = match[3]
		}
		if value == "" {
			value = match[4]
		}
		attributes[strings.ToLower(match[1])] = html.UnescapeString(strings.TrimSpace(value))
	}
	return attributes
}

func validRemoteURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.Host == "" {
		return nil, fmt.Errorf("invalid URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("credentials in URLs are not supported")
	}
	return parsed, nil
}

func normalizeWebsitePageURL(raw, defaultScheme string) string {
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return ""
	}
	if defaultScheme != "http" && defaultScheme != "https" {
		return ""
	}
	if strings.HasPrefix(normalized, "//") {
		normalized = defaultScheme + ":" + normalized
	} else if !strings.Contains(normalized, "://") {
		normalized = defaultScheme + "://" + normalized
	}
	parsed, err := validRemoteURL(normalized)
	if err != nil {
		return ""
	}
	return parsed.String()
}

func (a *app) fetchLimited(target, accept string, maxBytes int64, timeout time.Duration) ([]byte, string, int, error) {
	parsed, err := validRemoteURL(target)
	if err != nil {
		return nil, "", 0, err
	}
	req, err := http.NewRequest(http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, "", 0, err
	}
	req.Header.Set("User-Agent", "KISS-Startpage/2.7 (+icon-import)")
	req.Header.Set("Accept", accept)
	client := *a.client
	client.Timeout = timeout
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		_, err := validRemoteURL(req.URL.String())
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", resp.StatusCode, fmt.Errorf("remote source returned %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "", resp.StatusCode, err
	}
	if int64(len(raw)) > maxBytes {
		return nil, "", resp.StatusCode, fmt.Errorf("remote content is too large")
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0]))
	return raw, contentType, resp.StatusCode, nil
}

func (a *app) fetchLimitedWithRetry(target, accept string, maxBytes int64, timeout time.Duration, attempts int) ([]byte, string, int, error) {
	var (
		raw         []byte
		contentType string
		status      int
		err         error
	)
	for attempt := 0; attempt < attempts; attempt++ {
		raw, contentType, status, err = a.fetchLimited(target, accept, maxBytes, timeout)
		if err == nil || (status != http.StatusTooManyRequests && status < http.StatusInternalServerError) {
			return raw, contentType, status, err
		}
		if attempt+1 < attempts {
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
		}
	}
	return raw, contentType, status, err
}

func discoverFaviconURLs(pageURL string, raw []byte) []string {
	base, err := validRemoteURL(pageURL)
	if err != nil {
		return nil
	}
	type candidate struct {
		url      string
		priority int
	}
	candidates := []candidate{}
	for _, tag := range faviconLinkRe.FindAllString(string(raw), -1) {
		attributes := parseHTMLAttributes(tag)
		rel := strings.ToLower(attributes["rel"])
		href := strings.TrimSpace(attributes["href"])
		if href == "" || (!strings.Contains(rel, "icon") && !strings.Contains(rel, "apple-touch-icon")) {
			continue
		}
		reference, err := url.Parse(href)
		if err != nil {
			continue
		}
		resolved := base.ResolveReference(reference)
		if _, err := validRemoteURL(resolved.String()); err != nil {
			continue
		}
		priority := 20
		if strings.Contains(rel, "apple-touch-icon") {
			priority = 30
		}
		if strings.Contains(strings.ToLower(attributes["type"]), "svg") || strings.HasSuffix(strings.ToLower(resolved.Path), ".svg") {
			priority = 40
		}
		candidates = append(candidates, candidate{url: resolved.String(), priority: priority})
	}
	root := *base
	root.Path = "/favicon.ico"
	root.RawQuery = ""
	root.Fragment = ""
	candidates = append(candidates, candidate{url: root.String(), priority: 1})
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].priority > candidates[j].priority })
	result := []string{}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		if !seen[candidate.url] {
			seen[candidate.url] = true
			result = append(result, candidate.url)
		}
	}
	return result
}

func (a *app) searchWebsiteIcons(contextURLs []string) ([]iconSearchResult, error) {
	results := []iconSearchResult{}
	seen := map[string]bool{}
	for _, rawPageURL := range contextURLs {
		if rawPageURL == "" {
			continue
		}
		pageURL, err := validRemoteURL(rawPageURL)
		if err != nil {
			continue
		}
		pageRaw, _, _, fetchErr := a.fetchLimited(pageURL.String(), "text/html,application/xhtml+xml", maxWebsitePageBytes, 6*time.Second)
		candidates := []string{}
		if fetchErr == nil {
			candidates = discoverFaviconURLs(pageURL.String(), pageRaw)
		} else {
			candidates = discoverFaviconURLs(pageURL.String(), nil)
		}
		if len(candidates) == 0 || seen[candidates[0]] {
			continue
		}
		seen[candidates[0]] = true
		results = append(results, iconSearchResult{
			Name:       pageURL.Hostname(),
			Reference:  candidates[0],
			Category:   "Website icon",
			PreviewURL: candidates[0],
			Source:     "website",
			Provider:   "website",
			SourceURL:  pageURL.String(),
		})
	}
	return results, nil
}

func metadataString(metadata map[string]wikimediaMetadataValue, key string) string {
	value, ok := metadata[key]
	if !ok || value.Value == nil {
		return ""
	}
	return strings.TrimSpace(asString(value.Value))
}

func wikimediaItemAllowed(info wikimediaImageInfo) bool {
	license := strings.ToLower(metadataString(info.ExtendedMetadata, "LicenseShortName"))
	attribution := strings.ToLower(metadataString(info.ExtendedMetadata, "AttributionRequired"))
	copyrighted := strings.ToLower(metadataString(info.ExtendedMetadata, "Copyrighted"))
	restrictions := strings.TrimSpace(metadataString(info.ExtendedMetadata, "Restrictions"))
	licenseAllowed := license == "public domain" || license == "cc0" || strings.HasPrefix(license, "cc0 ")
	return licenseAllowed && attribution == "false" && copyrighted == "false" && restrictions == ""
}

func (a *app) queryWikimedia(parameters url.Values) ([]wikimediaPage, error) {
	parameters.Set("action", "query")
	parameters.Set("format", "json")
	parameters.Set("formatversion", "2")
	target := strings.TrimRight(a.cfg.WikimediaAPIBase, "?") + "?" + parameters.Encode()
	resp, err := a.httpGet(target, 15*time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Wikimedia search unavailable (%d)", resp.StatusCode)
	}
	var parsed wikimediaQueryResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4*1024*1024)).Decode(&parsed); err != nil {
		return nil, err
	}
	return parsed.Query.Pages, nil
}

func (a *app) searchWikimediaIcons(query string, limit int) ([]iconSearchResult, error) {
	searchLimit := limit * 5
	if searchLimit < 25 {
		searchLimit = 25
	}
	if searchLimit > 50 {
		searchLimit = 50
	}
	parameters := url.Values{
		"generator":    {"search"},
		"gsrnamespace": {"6"},
		"gsrsearch":    {strings.TrimSpace(query) + " icon filetype:svg"},
		"gsrlimit":     {fmt.Sprintf("%d", searchLimit)},
		"prop":         {"imageinfo"},
		"iiprop":       {"url|mime|size|extmetadata"},
		"iiurlwidth":   {"256"},
	}
	pages, err := a.queryWikimedia(parameters)
	if err != nil {
		return nil, err
	}
	results := []iconSearchResult{}
	for _, page := range pages {
		if len(page.ImageInfo) == 0 || !wikimediaItemAllowed(page.ImageInfo[0]) {
			continue
		}
		info := page.ImageInfo[0]
		name := strings.TrimSuffix(strings.TrimPrefix(page.Title, "File:"), filepath.Ext(page.Title))
		results = append(results, iconSearchResult{
			Name:       titleFromSlug(name),
			Reference:  page.Title,
			Category:   "Public domain / CC0",
			PreviewURL: info.ThumbURL,
			Source:     "wikimedia",
			Provider:   "wikimedia",
			License:    metadataString(info.ExtendedMetadata, "LicenseShortName"),
			LicenseURL: metadataString(info.ExtendedMetadata, "LicenseUrl"),
			SourceURL:  info.DescriptionURL,
		})
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func (a *app) handleImportIcon(w http.ResponseWriter, r *http.Request, payload map[string]any) {
	if _, ok := a.requireAuth(w, r, true); !ok {
		return
	}
	provider := strings.ToLower(strings.TrimSpace(asString(payload["provider"])))
	reference := strings.TrimSpace(asString(payload["reference"]))
	format := strings.ToLower(strings.TrimSpace(asString(payload["format"])))
	if format != "png" {
		format = "svg"
	}
	if _, ok := iconProviderByID(provider); !ok || reference == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"message": "Invalid icon provider or reference."})
		return
	}
	var (
		imported map[string]any
		status   int
		err      error
	)
	switch provider {
	case "iconify":
		imported, status, err = a.fetchIconifyIconData(reference, format, "")
	case "selfhst":
		imported, status, err = a.fetchSelfhstIconData(reference, format)
	case "dashboard":
		imported, status, err = a.fetchDashboardIconData(reference)
	case "local":
		imported, status, err = a.fetchLocalIconData(reference)
	case "website":
		imported, status, err = a.fetchWebsiteIconData(reference)
	case "wikimedia":
		imported, status, err = a.fetchWikimediaIconData(reference)
	}
	if err != nil {
		writeJSON(w, status, map[string]any{"message": err.Error()})
		return
	}
	imported["ok"] = true
	imported["provider"] = provider
	writeJSON(w, http.StatusOK, imported)
}

func iconDataPayload(name, reference, filename, contentType string, raw []byte) map[string]any {
	return map[string]any{
		"name":        name,
		"reference":   reference,
		"icon":        filename,
		"iconData":    "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(raw),
		"format":      strings.TrimPrefix(filepath.Ext(filename), "."),
		"contentType": contentType,
	}
}

func safeImageContent(raw []byte, contentType, filename string) (string, error) {
	extension := strings.ToLower(filepath.Ext(filename))
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = strings.ToLower(strings.TrimSpace(strings.SplitN(mime.TypeByExtension(extension), ";", 2)[0]))
	}
	if extension == ".svg" || contentType == "image/svg+xml" || bytesLookLikeSVG(raw) {
		text := strings.ToLower(string(raw))
		for _, forbidden := range []string{"<script", "<foreignobject", "javascript:", "<!entity", "<!doctype", "onload=", "onerror=", "onclick=", `href="http`, `href='http`} {
			if strings.Contains(text, forbidden) {
				return "", fmt.Errorf("unsafe SVG content")
			}
		}
		return "image/svg+xml", nil
	}
	detected := strings.ToLower(strings.TrimSpace(strings.SplitN(http.DetectContentType(raw), ";", 2)[0]))
	allowed := map[string]bool{
		"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true,
		"image/x-icon": true, "image/vnd.microsoft.icon": true,
	}
	if allowed[contentType] {
		return contentType, nil
	}
	if allowed[detected] {
		return detected, nil
	}
	return "", fmt.Errorf("unsupported image type")
}

func bytesLookLikeSVG(raw []byte) bool {
	trimmed := strings.TrimSpace(strings.ToLower(string(raw)))
	return strings.HasPrefix(trimmed, "<svg") || (strings.HasPrefix(trimmed, "<?xml") && strings.Contains(trimmed, "<svg"))
}

func (a *app) fetchDashboardIconData(reference string) (map[string]any, int, error) {
	items, err := a.getDashboardIconIndex()
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	var selected *selfhstIcon
	for index := range items {
		if strings.EqualFold(items[index].Reference, reference) {
			selected = &items[index]
			break
		}
	}
	if selected == nil {
		return nil, http.StatusNotFound, fmt.Errorf("icon not found")
	}
	extension := "png"
	if selected.HasSVG {
		extension = "svg"
	} else if selected.HasWebP {
		extension = "webp"
	}
	target := fmt.Sprintf("%s/%s/%s.%s", strings.TrimRight(a.cfg.DashboardRawBase, "/"), extension, url.PathEscape(selected.Reference), extension)
	raw, contentType, _, err := a.fetchLimited(target, "image/*", maxImportedIconBytes, 15*time.Second)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	contentType, err = safeImageContent(raw, contentType, selected.Reference+"."+extension)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	payload := iconDataPayload(selected.Name, selected.Reference, selected.Reference+"."+extension, contentType, raw)
	payload["license"] = "Apache-2.0"
	payload["licenseUrl"] = "https://github.com/homarr-labs/dashboard-icons/blob/main/LICENSE"
	return payload, http.StatusOK, nil
}

func (a *app) resolveLocalIcon(reference string) (string, error) {
	name := strings.TrimSpace(reference)
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") || strings.HasPrefix(name, ".") || !allowedLocalIconExtension(filepath.Ext(name)) {
		return "", fmt.Errorf("invalid local icon")
	}
	root, err := filepath.Abs(a.cfg.PrivateIconsDir)
	if err != nil {
		return "", err
	}
	target, err := filepath.Abs(filepath.Join(root, name))
	if err != nil || (target != root && !strings.HasPrefix(target, root+string(os.PathSeparator))) {
		return "", fmt.Errorf("invalid local icon")
	}
	return target, nil
}

func (a *app) fetchLocalIconData(reference string) (map[string]any, int, error) {
	target, err := a.resolveLocalIcon(reference)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	raw, err := os.ReadFile(target)
	if os.IsNotExist(err) {
		return nil, http.StatusNotFound, fmt.Errorf("local icon not found")
	}
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if len(raw) > maxImportedIconBytes {
		return nil, http.StatusBadRequest, fmt.Errorf("local icon is too large")
	}
	contentType, err := safeImageContent(raw, "", reference)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	return iconDataPayload(titleFromSlug(strings.TrimSuffix(reference, filepath.Ext(reference))), reference, reference, contentType, raw), http.StatusOK, nil
}

func (a *app) fetchWebsiteIconData(reference string) (map[string]any, int, error) {
	parsed, err := validRemoteURL(reference)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	raw, contentType, _, err := a.fetchLimited(parsed.String(), "image/*", maxImportedIconBytes, 10*time.Second)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	filename := filepath.Base(parsed.Path)
	if filename == "" || filename == "." || filename == "/" {
		filename = "website-icon.ico"
	}
	contentType, err = safeImageContent(raw, contentType, filename)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	return iconDataPayload(parsed.Hostname(), parsed.String(), filename, contentType, raw), http.StatusOK, nil
}

func (a *app) fetchWikimediaIconData(reference string) (map[string]any, int, error) {
	parameters := url.Values{
		"titles":     {reference},
		"prop":       {"imageinfo"},
		"iiprop":     {"url|mime|size|extmetadata"},
		"iiurlwidth": {"256"},
	}
	pages, err := a.queryWikimedia(parameters)
	if err != nil {
		return nil, http.StatusBadGateway, err
	}
	if len(pages) != 1 || len(pages[0].ImageInfo) == 0 || !wikimediaItemAllowed(pages[0].ImageInfo[0]) {
		return nil, http.StatusNotFound, fmt.Errorf("eligible Wikimedia icon not found")
	}
	page := pages[0]
	info := page.ImageInfo[0]
	raw, contentType, _, originalErr := a.fetchLimitedWithRetry(info.URL, "image/svg+xml", maxImportedIconBytes, 15*time.Second, 2)
	filename := "wikimedia-" + fmt.Sprintf("%d", page.PageID) + ".svg"
	if originalErr == nil {
		contentType, originalErr = safeImageContent(raw, contentType, filename)
	}
	if originalErr != nil {
		raw, contentType, _, err = a.fetchLimitedWithRetry(info.ThumbURL, "image/png", maxImportedIconBytes, 15*time.Second, 3)
		if err != nil {
			return nil, http.StatusBadGateway, fmt.Errorf("Wikimedia SVG unavailable (%v) and thumbnail unavailable: %w", originalErr, err)
		}
		filename = "wikimedia-" + fmt.Sprintf("%d", page.PageID) + ".png"
		contentType, err = safeImageContent(raw, contentType, filename)
		if err != nil {
			return nil, http.StatusBadGateway, err
		}
	}
	name := strings.TrimSuffix(strings.TrimPrefix(page.Title, "File:"), filepath.Ext(page.Title))
	payload := iconDataPayload(titleFromSlug(name), page.Title, filename, contentType, raw)
	payload["license"] = metadataString(info.ExtendedMetadata, "LicenseShortName")
	payload["licenseUrl"] = metadataString(info.ExtendedMetadata, "LicenseUrl")
	payload["sourceUrl"] = info.DescriptionURL
	return payload, http.StatusOK, nil
}
