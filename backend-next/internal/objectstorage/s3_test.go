package objectstorage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"image"
	"image/png"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestUploadedImageVerificationRejectsSpoofedTypeTruncationAndDigest(t *testing.T) {
	var out bytes.Buffer
	if err := png.Encode(&out, image.NewRGBA(image.Rect(0, 0, 4, 5))); err != nil {
		t.Fatal(err)
	}
	data := out.Bytes()
	sum := md5.Sum(data)
	digest := base64.StdEncoding.EncodeToString(sum[:])
	v, err := VerifyImage(data, "image/png", digest, int64(len(data)))
	if err != nil || v.Width != 4 || v.Height != 5 || len(v.SHA256) != 64 {
		t.Fatal("valid PNG rejected", err)
	}
	if _, err = VerifyImage(data, "image/jpeg", digest, int64(len(data))); err == nil {
		t.Fatal("spoofed type accepted")
	}
	if _, err = VerifyImage(data, "image/png", strings.Repeat("a", 24), int64(len(data))); err == nil {
		t.Fatal("wrong digest accepted")
	}
	partial := data[:len(data)/2]
	sum = md5.Sum(partial)
	if _, err = VerifyImage(partial, "image/png", base64.StdEncoding.EncodeToString(sum[:]), int64(len(partial))); err == nil {
		t.Fatal("truncated PNG accepted")
	}
}

func TestUploadSignaturePinsObjectAndOverwritePrecondition(t *testing.T) {
	c, err := New(Config{Endpoint: "https://s3.example.invalid", Region: "test-region", Bucket: "test-bucket", Prefix: "deuterium-test/", AccessKey: "synthetic-access-key", SecretKey: "synthetic-secret-key-not-valid"})
	if err != nil {
		t.Fatal(err)
	}
	a, err := c.Upload(context.Background(), "deuterium-test/uploads/only-object.png", "image/png", "1B2M2Y8AsgTpgAmY7PhCfg==", 100, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(a.UploadURL)
	if err != nil {
		t.Fatal(err)
	}
	signed := u.Query().Get("X-Amz-SignedHeaders")
	for _, header := range []string{"content-md5", "if-none-match", "content-length"} {
		if !strings.Contains(signed, header) {
			t.Fatalf("required signed header missing: %s in %s", header, signed)
		}
	}
	if a.SignedHeaders["If-None-Match"] != "*" || a.Provider != "RAINYUN_S3" || a.Method != "PUT" || time.Until(a.ExpiresAt) > 5*time.Minute {
		t.Fatal("upload protection missing")
	}
	if _, err = c.Upload(context.Background(), "other-prefix/key", "image/png", "1B2M2Y8AsgTpgAmY7PhCfg==", 100, time.Now().Add(time.Minute)); err == nil {
		t.Fatal("key escaped owned prefix")
	}
}
