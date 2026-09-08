package httpapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReleaseSelectionSeparatesPackageChannelAndCompatibility(t *testing.T) {
	now := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	base := PublishedRelease{ReleaseID: "apk", Kind: "APK", PackageName: "com.deuterium.app.uilab", Channel: "stable", VersionCode: 20, VersionName: "2.0.0", MinAppVersionCode: 1, MaxAppVersionCode: 100, PublishedAt: now.Add(-time.Hour)}
	preview := base
	preview.ReleaseID, preview.Channel, preview.VersionCode = "preview", "preview", 30
	other := base
	other.ReleaseID, other.PackageName, other.VersionCode = "other", "com.other.app", 100
	future := base
	future.ReleaseID, future.VersionCode, future.PublishedAt = "future", 99, now.Add(time.Hour)
	res := base
	res.ReleaseID, res.Kind, res.VersionCode, res.ResourceVersion, res.MinAppVersionCode = "resource", "RESOURCES", 0, 2, 20
	releases := []PublishedRelease{preview, other, res, base, future}
	old := releaseStatus(releases, base.PackageName, "stable", 9, 0, now)
	if old["latestVersionCode"] != int64(20) || old["apkUpdate"].(*PublishedRelease).ReleaseID != "apk" || old["resourceUpdate"].(*PublishedRelease) != nil {
		t.Fatalf("incorrect old-client update selection: %#v", old)
	}
	current := releaseStatus(releases, base.PackageName, "stable", 20, 0, now)
	if current["apkUpdate"].(*PublishedRelease) != nil || current["resourceUpdate"].(*PublishedRelease).ReleaseID != "resource" {
		t.Fatalf("resource compatibility ignored: %#v", current)
	}
	installed := releaseStatus(releases, base.PackageName, "stable", 20, 2, now)
	if installed["hasUpdates"] != false || releases[0].ReleaseID != "preview" {
		t.Fatal("already installed release offered or shared manifest mutated")
	}
}

func TestReleaseManifestRejectsUntrustedArtifactMetadata(t *testing.T) {
	p := filepath.Join(t.TempDir(), "releases.json")
	valid := PublishedRelease{ReleaseID: "r1", Kind: "APK", VersionName: "2.0.0", VersionCode: 20, PackageName: "com.deuterium.app.uilab", ReleaseNotes: "更新", DownloadURL: "https://47.103.99.34/downloads/app.apk", SizeBytes: 100, SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", MinAppVersionCode: 1, MaxAppVersionCode: 100, PublishedAt: time.Now(), Channel: "stable"}
	write := func(v any) {
		b, _ := json.Marshal(v)
		if err := os.WriteFile(p, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write([]PublishedRelease{valid})
	if _, err := readReleaseManifest(p); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*PublishedRelease){
		func(r *PublishedRelease) { r.DownloadURL = "http://47.103.99.34/file.apk" },
		func(r *PublishedRelease) { r.DownloadURL = "https://user:password@example.org/file.apk" },
		func(r *PublishedRelease) { r.SHA256 = "bad" },
		func(r *PublishedRelease) { r.SizeBytes = 201 << 20 },
		func(r *PublishedRelease) { r.Channel = "hidden" },
	} {
		broken := valid
		change(&broken)
		write([]PublishedRelease{broken})
		if _, err := readReleaseManifest(p); err == nil {
			t.Fatalf("accepted invalid artifact: %#v", broken)
		}
	}
	write([]PublishedRelease{valid, valid})
	if _, err := readReleaseManifest(p); err == nil {
		t.Fatal("duplicate release ID accepted")
	}
}
