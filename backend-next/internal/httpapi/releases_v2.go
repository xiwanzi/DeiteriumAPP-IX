package httpapi

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PublishedRelease is an operator-verified artifact. It never accepts a client
// supplied URL or hash and never fabricates an update from the caller's version.
type PublishedRelease struct {
	ReleaseID         string    `json:"releaseId"`
	Kind              string    `json:"kind"`
	VersionName       string    `json:"versionName"`
	VersionCode       int64     `json:"versionCode"`
	ResourceVersion   int64     `json:"resourceVersion"`
	PackageName       string    `json:"packageName"`
	ReleaseNotes      string    `json:"releaseNotes"`
	DownloadURL       string    `json:"downloadUrl"`
	SizeBytes         int64     `json:"sizeBytes"`
	SHA256            string    `json:"sha256"`
	MinAppVersionCode int64     `json:"minAppVersionCode"`
	MaxAppVersionCode int64     `json:"maxAppVersionCode"`
	PublishedAt       time.Time `json:"publishedAt"`
	Channel           string    `json:"channel"`
}

func readReleaseManifest(path string) ([]PublishedRelease, error) {
	if path == "" {
		return []PublishedRelease{}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.New("release manifest unavailable")
	}
	defer f.Close()
	decoder := json.NewDecoder(io.LimitReader(f, 1<<20))
	decoder.DisallowUnknownFields()
	var releases []PublishedRelease
	if decoder.Decode(&releases) != nil || len(releases) > 1000 {
		return nil, errors.New("invalid release manifest")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return nil, errors.New("invalid release manifest suffix")
	}
	seen := map[string]bool{}
	for _, r := range releases {
		u, err := url.Parse(r.DownloadURL)
		digest, hashErr := hex.DecodeString(r.SHA256)
		limit := int64(200 << 20)
		if r.Kind == "RESOURCES" {
			limit = 20 << 20
		}
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" ||
			r.ReleaseID == "" || seen[r.ReleaseID] || r.PackageName == "" || r.VersionName == "" ||
			(r.Channel != "stable" && r.Channel != "preview") ||
			(r.Kind != "APK" && r.Kind != "RESOURCES") || r.SizeBytes < 1 || r.SizeBytes > limit ||
			hashErr != nil || len(digest) != 32 || strings.ToLower(r.SHA256) != r.SHA256 ||
			r.MinAppVersionCode < 0 || r.MaxAppVersionCode < r.MinAppVersionCode ||
			(r.Kind == "APK" && r.VersionCode < 1) || (r.Kind == "RESOURCES" && r.ResourceVersion < 1) || r.PublishedAt.IsZero() {
			return nil, errors.New("invalid published release")
		}
		seen[r.ReleaseID] = true
	}
	return releases, nil
}

func releaseStatus(releases []PublishedRelease, packageName, channel string, appVersion, resourceVersion int64, now time.Time) map[string]any {
	var apk, resource *PublishedRelease
	var latestCode int64
	latestName := ""
	// Sort a copy: concurrent requests share the immutable manifest.
	ordered := append([]PublishedRelease(nil), releases...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].VersionCode != ordered[j].VersionCode {
			return ordered[i].VersionCode > ordered[j].VersionCode
		}
		return ordered[i].ResourceVersion > ordered[j].ResourceVersion
	})
	for i := range ordered {
		r := &ordered[i]
		if r.PackageName != packageName || r.Channel != channel || r.PublishedAt.After(now) {
			continue
		}
		if r.Kind == "APK" && r.VersionCode > latestCode {
			latestCode, latestName = r.VersionCode, r.VersionName
		}
		if appVersion < r.MinAppVersionCode || appVersion > r.MaxAppVersionCode {
			continue
		}
		if r.Kind == "APK" && r.VersionCode > appVersion && apk == nil {
			apk = r
		}
		if r.Kind == "RESOURCES" && r.ResourceVersion > resourceVersion && (resource == nil || r.ResourceVersion > resource.ResourceVersion) {
			resource = r
		}
	}
	message := "已是最新版本"
	if latestCode == 0 {
		message = "暂无可用更新"
	}
	hasUpdates := apk != nil || resource != nil
	if hasUpdates {
		message = "有更新可用"
	}
	return map[string]any{"latest": !hasUpdates, "message": message, "latestVersionCode": latestCode, "latestVersionName": latestName,
		"serverNow": now.UTC(), "hasUpdates": hasUpdates, "apkUpdate": apk, "resourceUpdate": resource}
}

func (s *Server) registerReleasesV2(mux *http.ServeMux) {
	releases, manifestError := readReleaseManifest(os.Getenv("DEUTERIUM_RELEASE_MANIFEST"))
	mux.HandleFunc("GET /api/v1/app/update-check", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		version, err := strconv.ParseInt(q.Get("versionCode"), 10, 32)
		resource, resourceErr := strconv.ParseInt(defaultValue(q.Get("resourceVersion"), "0"), 10, 32)
		channel := defaultValue(q.Get("channel"), "stable")
		packageName := defaultValue(q.Get("packageName"), "com.deuterium.app.uilab")
		if err != nil || resourceErr != nil || version < 0 || resource < 0 || len(packageName) > 200 ||
			(channel != "stable" && channel != "preview") {
			failure(w, r, 400, "INVALID_REQUEST", "版本或发布频道不正确。")
			return
		}
		if manifestError != nil {
			failure(w, r, 503, "UPDATE_UNAVAILABLE", "更新信息暂不可用，请稍后重试。")
			return
		}
		success(w, r, releaseStatus(releases, packageName, channel, version, resource, time.Now()))
	})
}

func defaultValue(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
