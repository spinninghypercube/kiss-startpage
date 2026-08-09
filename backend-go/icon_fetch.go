package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

func (a *app) httpGet(target string, timeout time.Duration) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "kiss-startpage-go/0.1 (+selfhst-icons)")
	req.Header.Set("Accept", "application/json, image/svg+xml, image/png;q=0.9, */*;q=0.8")
	client := *a.client
	client.Timeout = timeout
	return client.Do(req)
}

func normalizeBoolFlag(v any) bool {
	s := strings.ToLower(strings.TrimSpace(asString(v)))
	return truthyFlagValues[s]
}

func (a *app) getSelfhstIconIndex() ([]selfhstIcon, error) {
	now := time.Now().Unix()
	a.iconMu.Lock()
	cache := a.iconIdx
	a.iconMu.Unlock()
	if len(cache.Items) > 0 && now-cache.FetchedAt < int64(a.cfg.IconIndexTTL) {
		return cache.Items, nil
	}
	resp, err := a.httpGet(a.cfg.SelfhstIndexURL, 20*time.Second)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("icon index source error (%d)", resp.StatusCode)
	}
	var rows []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	items := make([]selfhstIcon, 0, len(rows))
	for _, row := range rows {
		ref := strings.TrimSpace(asString(row["Reference"]))
		if ref == "" {
			continue
		}
		name := strings.TrimSpace(asString(row["Name"]))
		if name == "" {
			name = ref
		}
		items = append(items, selfhstIcon{
			Name:      name,
			Reference: ref,
			Category:  strings.TrimSpace(asString(row["Category"])),
			Tags:      strings.TrimSpace(asString(row["Tags"])),
			HasSVG:    normalizeBoolFlag(row["SVG"]),
			HasPNG:    normalizeBoolFlag(row["PNG"]),
			HasWebP:   normalizeBoolFlag(row["WebP"]),
			HasLight:  normalizeBoolFlag(row["Light"]),
			HasDark:   normalizeBoolFlag(row["Dark"]),
		})
	}
	a.iconMu.Lock()
	a.iconIdx = iconCache{FetchedAt: now, Items: items}
	a.iconMu.Unlock()
	return items, nil
}

func (a *app) searchSelfhstIcons(query string, limit int) ([]iconSearchResult, error) {
	items, err := a.getSelfhstIconIndex()
	if err != nil {
		return nil, err
	}
	nq := strings.ToLower(strings.TrimSpace(query))
	tokens := strings.Fields(nq)
	results := make([]iconSearchResult, 0, len(items))
	for _, item := range items {
		name := strings.ToLower(item.Name)
		ref := strings.ToLower(item.Reference)
		cat := strings.ToLower(item.Category)
		tags := strings.ToLower(item.Tags)
		haystack := name + " " + ref + " " + cat + " " + tags
		if nq != "" && !strings.Contains(haystack, nq) {
			continue
		}
		score := 0
		if ref == nq {
			score += 1200
		} else if strings.HasPrefix(ref, nq) {
			score += 900
		}
		if name == nq {
			score += 1000
		} else if strings.HasPrefix(name, nq) {
			score += 800
		}
		if strings.Contains(name, nq) {
			score += 500
		}
		if strings.Contains(ref, nq) {
			score += 450
		}
		if strings.Contains(tags, nq) {
			score += 220
		}
		if strings.Contains(cat, nq) {
			score += 120
		}
		for _, tok := range tokens {
			if strings.Contains(name, tok) {
				score += 80
			}
			if strings.Contains(ref, tok) {
				score += 70
			}
			if strings.Contains(tags, tok) {
				score += 35
			}
		}
		previewExt := ""
		if item.HasSVG {
			previewExt = "svg"
		} else if item.HasPNG {
			previewExt = "png"
		}
		previewURL := ""
		if previewExt != "" {
			previewURL = fmt.Sprintf("%s/%s/%s.%s", a.cfg.SelfhstRawBase, previewExt, url.PathEscape(item.Reference), previewExt)
		}
		results = append(results, iconSearchResult{
			Score:      score,
			Name:       item.Name,
			Reference:  item.Reference,
			Category:   item.Category,
			Tags:       item.Tags,
			HasSVG:     item.HasSVG,
			HasPNG:     item.HasPNG,
			HasWebP:    item.HasWebP,
			HasLight:   item.HasLight,
			HasDark:    item.HasDark,
			PreviewURL: previewURL,
		})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if strings.ToLower(results[i].Name) != strings.ToLower(results[j].Name) {
			return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
		}
		return strings.ToLower(results[i].Reference) < strings.ToLower(results[j].Reference)
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (a *app) fetchSelfhstIconData(reference, preferFormat string) (map[string]any, int, error) {
	items, err := a.getSelfhstIconIndex()
	if err != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("failed to import icon: %v", err)
	}
	var item *selfhstIcon
	for i := range items {
		if strings.EqualFold(items[i].Reference, reference) {
			item = &items[i]
			break
		}
	}
	if item == nil {
		return nil, http.StatusNotFound, fmt.Errorf("Icon not found.")
	}
	formats := []string{}
	if preferFormat == "png" {
		if item.HasPNG {
			formats = append(formats, "png")
		}
		if item.HasSVG {
			formats = append(formats, "svg")
		}
	} else {
		if item.HasSVG {
			formats = append(formats, "svg")
		}
		if item.HasPNG {
			formats = append(formats, "png")
		}
	}
	if len(formats) == 0 {
		return nil, http.StatusBadRequest, fmt.Errorf("Selected icon does not have a supported format.")
	}
	var lastErr error
	for _, ext := range formats {
		target := fmt.Sprintf("%s/%s/%s.%s", strings.TrimRight(a.cfg.SelfhstRawBase, "/"), ext, url.PathEscape(item.Reference), ext)
		resp, err := a.httpGet(target, 20*time.Second)
		if err != nil {
			lastErr = err
			continue
		}
		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode == http.StatusNotFound {
			lastErr = fmt.Errorf("404")
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, http.StatusBadGateway, fmt.Errorf("Icon source error (%d).", resp.StatusCode)
		}
		if len(raw) == 0 {
			continue
		}
		contentType := strings.ToLower(strings.TrimSpace(strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0]))
		if contentType == "" {
			if ext == "svg" {
				contentType = "image/svg+xml"
			} else {
				contentType = "image/png"
			}
		}
		return map[string]any{
			"name":        item.Name,
			"reference":   item.Reference,
			"icon":        fmt.Sprintf("%s.%s", item.Reference, ext),
			"iconData":    "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(raw),
			"format":      ext,
			"contentType": contentType,
		}, http.StatusOK, nil
	}
	if lastErr != nil {
		return nil, http.StatusBadGateway, fmt.Errorf("Failed to import icon: %v", lastErr)
	}
	return nil, http.StatusBadGateway, fmt.Errorf("Failed to import icon: unknown error")
}

func normalizeIconifyName(iconName, sourceHint string) (value, prefix, name string, sourceID string, err error) {
	value = strings.ToLower(strings.TrimSpace(iconName))
	value = strings.Join(strings.Fields(value), "")
	value = regexp.MustCompile(`:+`).ReplaceAllString(value, ":")
	sourceHint = strings.ToLower(strings.TrimSpace(sourceHint))
	if !strings.Contains(value, ":") {
		source, ok := iconSourceByID(sourceHint)
		if !ok || source.Provider != "iconify" {
			err = fmt.Errorf("Unsupported Iconify icon set.")
			return
		}
		value = source.Prefix + ":" + value
	}
	if !iconifyNameRe.MatchString(value) {
		err = fmt.Errorf("Invalid icon name.")
		return
	}
	parts := strings.SplitN(value, ":", 2)
	prefix, name = parts[0], parts[1]
	source, ok := iconSourceByPrefix(prefix)
	if !ok {
		err = fmt.Errorf("Unsupported Iconify icon set.")
		return
	}
	if sourceHint != "" && sourceHint != source.ID {
		err = fmt.Errorf("Icon source does not match icon name.")
		return
	}
	sourceID = source.ID
	return
}

func (a *app) fetchIconifyIconData(iconName, preferFormat, sourceHint string) (map[string]any, int, error) {
	value, prefix, name, sourceID, err := normalizeIconifyName(iconName, sourceHint)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	fmtWanted := preferFormat
	if fmtWanted != "png" {
		fmtWanted = "svg"
	}
	formats := []string{fmtWanted}
	if fmtWanted == "png" {
		formats = append(formats, "svg")
	}
	var lastStatus int
	var lastErr error
	for _, ext := range formats {
		target := fmt.Sprintf("%s/%s/%s.%s", strings.TrimRight(a.cfg.IconifyAPIBase, "/"), url.PathEscape(prefix), url.PathEscape(name), ext)
		resp, err := a.httpGet(target, 20*time.Second)
		if err != nil {
			lastErr = err
			lastStatus = http.StatusBadGateway
			continue
		}
		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			lastStatus = http.StatusBadGateway
			continue
		}
		if resp.StatusCode == http.StatusNotFound && ext == "png" {
			lastErr = fmt.Errorf("404")
			lastStatus = http.StatusBadGateway
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, http.StatusBadGateway, fmt.Errorf("Icon source error (%d).", resp.StatusCode)
		}
		if len(raw) == 0 {
			lastErr = fmt.Errorf("empty icon data")
			lastStatus = http.StatusBadGateway
			continue
		}
		contentType := strings.ToLower(strings.TrimSpace(strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0]))
		if contentType == "" {
			if ext == "svg" {
				contentType = "image/svg+xml"
			} else {
				contentType = "image/png"
			}
		}
		label := strings.Title(strings.ReplaceAll(name, "-", " "))
		return map[string]any{
			"name":        label,
			"reference":   value,
			"source":      sourceID,
			"icon":        fmt.Sprintf("%s.%s", name, ext),
			"iconData":    "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(raw),
			"format":      ext,
			"contentType": contentType,
		}, http.StatusOK, nil
	}
	if lastErr != nil {
		return nil, lastStatus, fmt.Errorf("Failed to import icon: %v", lastErr)
	}
	return nil, http.StatusBadGateway, fmt.Errorf("Failed to import icon.")
}
