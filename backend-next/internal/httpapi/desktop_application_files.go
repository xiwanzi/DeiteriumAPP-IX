package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const desktopApplicationExpandedLimit int64 = 4 * 1024 * 1024 * 1024

type desktopApplicationFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

var applicationDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)
var applicationReservedName = regexp.MustCompile(`(?i)^(CON|PRN|AUX|NUL|COM[1-9¹²³]|LPT[1-9¹²³])(?:\..*)?$`)

func desktopApplicationOwnedPath(path string) bool {
	if path == "" || utf8.RuneCountInString(path) > 230 || !utf8.ValidString(path) || strings.ContainsAny(path, `\<>:"|?*`) {
		return false
	}
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") || applicationReservedName.MatchString(part) {
			return false
		}
		for _, r := range part {
			if r < 32 {
				return false
			}
		}
	}
	if strings.EqualFold(path, "DLauncher.exe") || strings.EqualFold(path, "DLauncher/certifi/cacert.pem") {
		return true
	}
	if len(parts) < 2 || parts[0] != "DLauncher" {
		return false
	}
	for _, blocked := range []string{"java", "launcher.json", "bundle.json", "settings.json", "accounts", "accounts.json", "community", "cache", "logs", ".update", "application-files.json", ".minecraft", ".dlauncher"} {
		if strings.EqualFold(parts[1], blocked) {
			return false
		}
	}
	for _, part := range parts {
		for _, suffix := range []string{".local.json", ".dpapi", ".env", ".log", ".pem", ".key"} {
			if strings.HasSuffix(strings.ToLower(part), suffix) {
				return false
			}
		}
	}
	return true
}

func desktopApplicationFiles(raw []byte, version string) ([]desktopApplicationFile, error) {
	invalid := errors.New("invalid launcher file manifest")
	if !utf8.Valid(raw) || int64(len(raw)) > desktopApplicationManifestLimit {
		return nil, invalid
	}
	fields, err := applicationFields(raw, "schemaVersion", "version", "files")
	if err != nil {
		return nil, invalid
	}
	var manifest struct {
		SchemaVersion int                      `json:"schemaVersion"`
		Version       string                   `json:"version"`
		Files         []desktopApplicationFile `json:"files"`
	}
	var entries []json.RawMessage
	if strictApplicationJSON(raw, &manifest) != nil || json.Unmarshal(fields["files"], &entries) != nil || manifest.SchemaVersion != 1 || manifest.Version != version || len(entries) < 2 || len(entries) > 20000 {
		return nil, invalid
	}
	seen, sizes := map[string]bool{}, map[string]int64{}
	var total int64
	for i, item := range manifest.Files {
		if _, err := applicationFields(entries[i], "path", "size", "sha256"); err != nil || !desktopApplicationOwnedPath(item.Path) || seen[strings.ToLower(item.Path)] || item.Size < 0 || item.Size > desktopApplicationExpandedLimit || !applicationDigest.MatchString(item.SHA256) {
			return nil, invalid
		}
		if size, ok := sizes[item.SHA256]; ok && size != item.Size {
			return nil, invalid
		}
		sizes[item.SHA256] = item.Size
		seen[strings.ToLower(item.Path)] = true
		total += item.Size
		if total > desktopApplicationExpandedLimit {
			return nil, invalid
		}
	}
	if !seen["dlauncher.exe"] || !seen["dlauncher/updater.exe"] {
		return nil, invalid
	}
	for path := range seen {
		parts := strings.Split(path, "/")
		for i := 1; i < len(parts); i++ {
			if seen[strings.Join(parts[:i], "/")] {
				return nil, invalid
			}
		}
	}
	return manifest.Files, nil
}

type desktopApplicationObject struct {
	item    desktopApplicationFile
	entries []*zip.File
}

func (d *desktopRuntime) prepareApplicationFiles(ctx context.Context, release desktopApplicationRelease, archive *os.File, size int64) error {
	reader, err := zip.NewReader(archive, size)
	if err != nil || len(reader.File) > 20001 {
		return errors.New("invalid complete launcher ZIP")
	}
	entries := map[string]*zip.File{}
	for _, entry := range reader.File {
		key := strings.ToLower(entry.Name)
		if entries[key] != nil || entry.Flags&1 != 0 || entry.FileInfo().Mode()&os.ModeType != 0 || (entry.Name != "update-manifest.json" && !desktopApplicationOwnedPath(entry.Name)) {
			return errors.New("unsafe or duplicate launcher ZIP entry")
		}
		entries[key] = entry
	}
	manifest := entries["update-manifest.json"]
	if manifest == nil || manifest.UncompressedSize64 != uint64(release.Manifest.Size) {
		return errors.New("launcher manifest size mismatch")
	}
	stream, err := manifest.Open()
	if err != nil {
		return err
	}
	raw, err := io.ReadAll(io.LimitReader(stream, release.Manifest.Size+1))
	stream.Close()
	digest := sha256.Sum256(raw)
	if err != nil || int64(len(raw)) != release.Manifest.Size || hex.EncodeToString(digest[:]) != release.Manifest.SHA256 {
		return errors.New("launcher manifest digest mismatch")
	}
	files, err := desktopApplicationFiles(raw, release.Version)
	if err != nil || len(entries) != len(files)+1 {
		return errors.New("launcher package does not match manifest")
	}
	objects := map[string]*desktopApplicationObject{}
	for _, item := range files {
		entry := entries[strings.ToLower(item.Path)]
		if entry == nil || entry.Name != item.Path || entry.UncompressedSize64 != uint64(item.Size) {
			return errors.New("launcher file size or path mismatch")
		}
		object := objects[item.SHA256]
		if object == nil {
			object = &desktopApplicationObject{item: item}
			objects[item.SHA256] = object
		}
		object.entries = append(object.entries, entry)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan *desktopApplicationObject)
	var workers sync.WaitGroup
	var once sync.Once
	var firstError error
	for i := 0; i < 4; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for object := range jobs {
				if err := d.prepareApplicationObject(ctx, object); err != nil {
					once.Do(func() { firstError = err; cancel() })
				}
			}
		}()
	}
send:
	for _, object := range objects {
		select {
		case jobs <- object:
		case <-ctx.Done():
			break send
		}
	}
	close(jobs)
	workers.Wait()
	if firstError != nil {
		return firstError
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	md5sum := md5.Sum(raw) // S3 transport checksum; trust remains signed SHA-256.
	_, err = d.objects.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(d.bucket), Key: aws.String(desktopApplicationManifestKey(release.Manifest.SHA256)), Body: bytes.NewReader(raw), ContentLength: aws.Int64(int64(len(raw))), ContentType: aws.String("application/json"), ContentMD5: aws.String(base64.StdEncoding.EncodeToString(md5sum[:])), CacheControl: aws.String("public, max-age=31536000, immutable"), Metadata: map[string]string{"sha256": release.Manifest.SHA256}})
	return err
}

func (d *desktopRuntime) prepareApplicationObject(ctx context.Context, object *desktopApplicationObject) error {
	item := object.item
	key := desktopApplicationFileKey(item.SHA256)
	head, err := d.objects.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(d.bucket), Key: aws.String(key)})
	reuse := err == nil && aws.ToInt64(head.ContentLength) == item.Size && head.Metadata["sha256"] == item.SHA256
	var temporary *os.File
	if !reuse {
		temporary, err = os.CreateTemp("", "deuterium-launcher-file-*")
		if err != nil {
			return err
		}
		defer func() { temporary.Close(); os.Remove(temporary.Name()) }()
	}
	transportHash := md5.New()
	for index, entry := range object.entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		stream, err := entry.Open()
		if err != nil {
			return err
		}
		digest := sha256.New()
		var destination io.Writer = digest
		if !reuse && index == 0 {
			destination = io.MultiWriter(digest, transportHash, temporary)
		}
		size, err := io.Copy(destination, io.LimitReader(stream, item.Size+1))
		stream.Close()
		if err != nil || size != item.Size || hex.EncodeToString(digest.Sum(nil)) != item.SHA256 {
			return errors.New("launcher content hash mismatch")
		}
	}
	if reuse {
		return nil
	}
	if _, err = temporary.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err = d.objects.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(d.bucket), Key: aws.String(key), Body: temporary, ContentLength: aws.Int64(item.Size), ContentType: aws.String("application/octet-stream"), ContentMD5: aws.String(base64.StdEncoding.EncodeToString(transportHash.Sum(nil))), CacheControl: aws.String("public, max-age=31536000, immutable"), Metadata: map[string]string{"sha256": item.SHA256}})
	return err
}
