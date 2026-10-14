package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type applicationBundleFixture struct {
	legacy, incremental desktopApplicationRelease
	public              ed25519.PublicKey
	private             ed25519.PrivateKey
	archive             []byte
	manifest            []byte
}

func applicationSHA(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func newApplicationBundle(t *testing.T) *applicationBundleFixture {
	t.Helper()
	f := &applicationBundleFixture{}
	f.public, f.private, _ = ed25519.GenerateKey(rand.Reader)
	values := map[string][]byte{"DLauncher.exe": []byte("launcher fixture"), "DLauncher/Updater.exe": []byte("updater fixture"), "DLauncher/runtime.dll": []byte("shared runtime"), "DLauncher/also-runtime.dll": []byte("shared runtime")}
	files := []desktopApplicationFile{}
	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	for path, raw := range values {
		files = append(files, desktopApplicationFile{Path: path, Size: int64(len(raw)), SHA256: applicationSHA(raw)})
		entry, _ := w.Create(path)
		entry.Write(raw)
	}
	f.manifest, _ = json.Marshal(map[string]any{"schemaVersion": 1, "version": "0.6.10", "files": files})
	entry, _ := w.Create("update-manifest.json")
	entry.Write(f.manifest)
	w.Close()
	f.archive = archive.Bytes()
	f.legacy = desktopApplicationRelease{SchemaVersion: 1, Product: "DLauncher", Platform: "windows-x64", Version: "0.6.10", VersionCode: 6010, UpdaterProtocol: 1, Notes: "fixture", PublishedAt: "2026-10-15T00:00:00Z"}
	f.legacy.Package = desktopApplicationArtifact{Size: int64(len(f.archive)), SHA256: applicationSHA(f.archive)}
	f.legacy.Package.URL = "https://objects.example/" + desktopApplicationObjectKey(f.legacy)
	f.incremental = f.legacy
	f.incremental.UpdaterProtocol = 2
	f.incremental.Manifest = &desktopApplicationArtifact{Size: int64(len(f.manifest)), SHA256: applicationSHA(f.manifest), URL: "https://objects.example/" + desktopApplicationManifestKey(applicationSHA(f.manifest))}
	return f
}

func (f *applicationBundleFixture) sign(value desktopApplicationRelease) json.RawMessage {
	payload, _ := json.Marshal(value)
	raw, _ := json.Marshal(desktopApplicationEnvelope{Payload: base64.StdEncoding.EncodeToString(payload), Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(f.private, payload))})
	return raw
}

func (f *applicationBundleFixture) publication() json.RawMessage {
	raw, _ := json.Marshal(map[string]any{"schemaVersion": 2, "legacy": f.sign(f.legacy), "incremental": f.sign(f.incremental)})
	return raw
}

type applicationObjectFile struct {
	path, digest string
	size         int64
}

type applicationObjectsFixture struct {
	directory string
	mu        sync.Mutex
	objects   map[string]applicationObjectFile
	puts      map[string]int
	fail      atomic.Bool
	server    *httptest.Server
}

func newApplicationObjects(t *testing.T) *applicationObjectsFixture {
	t.Helper()
	f := &applicationObjectsFixture{directory: t.TempDir(), objects: map[string]applicationObjectFile{}, puts: map[string]int{}}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/fixture/")
		if r.Method == "PUT" {
			if f.fail.Load() {
				http.Error(w, "fixture unavailable", 403)
				return
			}
			file, err := os.CreateTemp(f.directory, "object-*")
			if err != nil {
				t.Error(err)
				w.WriteHeader(500)
				return
			}
			digest, transport := sha256.New(), md5.New()
			size, err := io.Copy(io.MultiWriter(file, digest, transport), r.Body)
			file.Close()
			hash := hex.EncodeToString(digest.Sum(nil))
			if err != nil || size != r.ContentLength || base64.StdEncoding.EncodeToString(transport.Sum(nil)) != r.Header.Get("Content-MD5") || hash != r.Header.Get("X-Amz-Meta-Sha256") {
				t.Error("invalid uploaded content/checksum", key, err)
				w.WriteHeader(400)
				return
			}
			f.mu.Lock()
			f.objects[key] = applicationObjectFile{path: file.Name(), digest: hash, size: size}
			f.puts[key]++
			f.mu.Unlock()
			w.WriteHeader(200)
			return
		}
		f.mu.Lock()
		object, ok := f.objects[key]
		f.mu.Unlock()
		if !ok {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("Content-Length", strconv.FormatInt(object.size, 10))
		w.Header().Set("X-Amz-Meta-Sha256", object.digest)
		if r.Method == "GET" {
			file, err := os.Open(object.path)
			if err != nil {
				t.Error(err)
				w.WriteHeader(500)
				return
			}
			defer file.Close()
			io.Copy(w, file)
		} else if r.Method != "HEAD" {
			w.WriteHeader(405)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *applicationObjectsFixture) runtime() *desktopRuntime {
	return &desktopRuntime{bucket: "fixture", publicURL: "https://objects.example", objects: s3.New(s3.Options{Region: "us-east-1", BaseEndpoint: aws.String(f.server.URL), UsePathStyle: true, Credentials: credentials.NewStaticCredentialsProvider("fixture", "fixture", ""), RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired, ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired})}
}

func (f *applicationObjectsFixture) seed(key, path string, size int64, digest string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = applicationObjectFile{path: path, size: size, digest: digest}
}

func TestDesktopApplicationDualProtocolSignatureAndPairing(t *testing.T) {
	f := newApplicationBundle(t)
	if value, err := verifyDesktopPublication(f.publication(), f.public); err != nil || value.UpdaterProtocol != 2 {
		t.Fatal("valid dual publication rejected", err)
	}
	legacy, incremental, err := desktopApplicationEnvelopes(f.publication())
	if err != nil || !bytes.Equal(legacy, f.sign(f.legacy)) || !bytes.Equal(incremental, f.sign(f.incremental)) {
		t.Fatal("protocol envelopes not preserved", err)
	}
	if _, err := verifyDesktopPublication(f.sign(f.legacy), f.public); err != nil {
		t.Fatal("legacy publication rejected", err)
	}
	if _, err := verifyDesktopPublication(f.sign(f.incremental), f.public); err == nil {
		t.Fatal("publication without legacy compatibility accepted")
	}
	for _, mutate := range []func(*desktopApplicationRelease){
		func(v *desktopApplicationRelease) { v.Notes = "different" },
		func(v *desktopApplicationRelease) { v.Version = "0.6.11"; v.VersionCode = 6011 },
		func(v *desktopApplicationRelease) { v.Package.Size++ },
		func(v *desktopApplicationRelease) { v.Manifest.URL = "https://elsewhere.test/manifest.json" },
	} {
		candidate := newApplicationBundle(t)
		mutate(&candidate.incremental)
		if _, err := verifyDesktopPublication(candidate.publication(), candidate.public); err == nil {
			t.Fatal("mismatched publication accepted")
		}
	}
}

func TestDesktopApplicationFileOwnershipAndDuplicates(t *testing.T) {
	for _, path := range []string{"../outside", "DLauncher/a:stream", "DLauncher/CON", ".minecraft/options.txt", "DLauncher/java/bin/java.exe", "DLauncher/launcher.json", "DLauncher/accounts/token.dpapi", "DLauncher/.update/task.json", "DLauncher/x/../secret"} {
		if desktopApplicationOwnedPath(path) {
			t.Fatal("private/unsafe path accepted", path)
		}
	}
	f := newApplicationBundle(t)
	var manifest map[string]any
	json.Unmarshal(f.manifest, &manifest)
	files := manifest["files"].([]any)
	manifest["files"] = append(files, files[0])
	raw, _ := json.Marshal(manifest)
	if _, err := desktopApplicationFiles(raw, "0.6.10"); err == nil {
		t.Fatal("duplicate manifest path accepted")
	}
}

func TestDesktopApplicationPreparesAndReusesVerifiedContentObjects(t *testing.T) {
	f, objects := newApplicationBundle(t), newApplicationObjects(t)
	path := filepath.Join(t.TempDir(), "application.zip")
	os.WriteFile(path, f.archive, 0600)
	archive, _ := os.Open(path)
	defer archive.Close()
	runtime := objects.runtime()
	for iteration := 0; iteration < 2; iteration++ {
		if err := runtime.prepareApplicationFiles(context.Background(), f.incremental, archive, int64(len(f.archive))); err != nil {
			t.Fatal(err)
		}
	}
	objects.mu.Lock()
	defer objects.mu.Unlock()
	if len(objects.objects) != 4 { // Three unique file contents plus one manifest.
		t.Fatal("unexpected object count", len(objects.objects))
	}
	for key, count := range objects.puts {
		if strings.Contains(key, "/files/") && count != 1 {
			t.Fatal("unchanged content was uploaded again", key, count)
		}
	}
}
