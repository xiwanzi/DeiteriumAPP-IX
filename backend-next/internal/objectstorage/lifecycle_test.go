package objectstorage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestLifecycleCopyUsesCompatibleSourceAndReusesVerifiedTarget(t *testing.T) {
	payload := []byte("verified lifecycle image")
	digest := sha256.Sum256(payload)
	sha := hex.EncodeToString(digest[:])
	source := "deuterium-test/uploads/image.png"
	target := "deuterium-test/gc/image.png"
	var mu sync.Mutex
	files := map[string][]byte{source: payload}
	created := time.Now().UTC().Truncate(time.Second)
	copies := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		key := strings.TrimPrefix(r.URL.Path, "/test-bucket/")
		if r.Method == "PUT" {
			if r.Header.Get("X-Amz-Copy-Source") != "test-bucket/"+source {
				t.Errorf("incompatible copy source: %s", r.Header.Get("X-Amz-Copy-Source"))
				http.Error(w, "invalid", 400)
				return
			}
			if r.Header.Get("X-Amz-Copy-Source-If-Match") != "\"fixture-etag\"" {
				t.Error("copy precondition missing")
			}
			files[key] = append([]byte(nil), files[source]...)
			copies++
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprintf(w, "<CopyObjectResult><LastModified>%s</LastModified><ETag>fixture-etag</ETag></CopyObjectResult>", created.Format(time.RFC3339))
			return
		}
		value, ok := files[key]
		if !ok {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(404)
			fmt.Fprint(w, "<Error><Code>NoSuchKey</Code></Error>")
			return
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(value)))
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("ETag", "\"fixture-etag\"")
		w.Header().Set("Last-Modified", created.Format(http.TimeFormat))
		if r.Method == "GET" {
			w.Write(value)
		}
	}))
	defer server.Close()
	c := &Client{Config: Config{Bucket: "test-bucket", Prefix: "deuterium-test/"}}
	c.S3 = s3.NewFromConfig(aws.Config{Region: "test", Credentials: credentials.NewStaticCredentialsProvider("fixture", "fixture", ""), HTTPClient: server.Client()}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(server.URL)
		o.UsePathStyle = true
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})
	first, err := c.CopyVerified(context.Background(), source, target, sha)
	if err != nil || !first.Equal(created) {
		t.Fatal("copy failed", first, err)
	}
	second, err := c.CopyVerified(context.Background(), source, target, sha)
	if err != nil || !second.Equal(first) || copies != 1 {
		t.Fatal("retry copied again", second, err, copies)
	}
	mu.Lock()
	delete(files, source)
	mu.Unlock()
	if _, err = c.CopyVerified(context.Background(), source, target, sha); err != nil {
		t.Fatal("could not recover verified target after source disappeared", err)
	}
	mu.Lock()
	files[target] = []byte("tampered")
	mu.Unlock()
	if _, err = c.CopyVerified(context.Background(), source, target, sha); err == nil {
		t.Fatal("corrupt target accepted after source disappeared")
	}
	if _, err = c.CopyVerified(context.Background(), "another-project/private.png", target, sha); err == nil {
		t.Fatal("cross-prefix source accepted")
	}
}
