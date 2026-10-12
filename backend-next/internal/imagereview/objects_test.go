package imagereview

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
)

func TestReviewStaticHeadersAndAnimationSkip(t *testing.T) {
	var pngData bytes.Buffer
	png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	webp := append([]byte("RIFF\x00\x00\x00\x00WEBPVP8X\x0a\x00\x00\x00"), make([]byte, 10)...)
	animated := append([]byte(nil), webp...)
	animated[20] = 2
	apng := append([]byte(nil), pngData.Bytes()[:8]...)
	chunk := make([]byte, 20)
	binary.BigEndian.PutUint32(chunk[:4], 8)
	copy(chunk[4:], "acTL")
	apng = append(apng, chunk...)
	for _, tc := range []struct {
		b        []byte
		kind     string
		animated bool
	}{{pngData.Bytes(), "image/png", false}, {[]byte{255, 216, 255, 224}, "image/jpeg", false}, {webp, "image/webp", false}, {animated, "image/webp", true}, {apng, "image/png", true}, {[]byte("GIF89a"), "image/gif", true}} {
		kind, moving, e := inspectHeader(tc.b)
		if e != nil || kind != tc.kind || moving != tc.animated {
			t.Fatal(kind, moving, e)
		}
	}
	if _, _, e := inspectHeader([]byte("<html>not an image</html>")); !errors.Is(e, ErrInvalidImage) {
		t.Fatal(e)
	}
}
func TestReviewEvidenceCopyRestartAndDeleteCannotTouchSource(t *testing.T) {
	present := false
	copies, deletes := 0, 0
	id := strings.Repeat("a", 64)
	target := "deuterium-test/image-review/evidence/" + id + ".image"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, target) {
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(500)
			return
		}
		switch r.Method {
		case "HEAD":
			if !present {
				w.WriteHeader(404)
				return
			}
			w.Header().Set("Content-Length", "100")
			w.Header().Set("ETag", `"digest"`)
		case "PUT":
			copies++
			if r.Header.Get("X-Amz-Copy-Source") != "bucket/deuterium-test/chat-media/media_1/image.webp" || r.Header.Get("X-Amz-Copy-Source-If-Match") != `"digest"` {
				t.Error("copy source guard missing")
			}
			present = true
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<CopyObjectResult><ETag>"digest"</ETag></CopyObjectResult>`))
		case "DELETE":
			deletes++
			present = false
			w.WriteHeader(204)
		default:
			t.Error("unexpected object operation", r.Method)
			w.WriteHeader(500)
		}
	}))
	defer srv.Close()
	sdk := s3.NewFromConfig(aws.Config{Region: "test", Credentials: credentials.NewStaticCredentialsProvider("test", "test", "")}, func(o *s3.Options) { o.BaseEndpoint = aws.String(srv.URL); o.UsePathStyle = true })
	objects := &S3Objects{Client: &objectstorage.Client{Config: objectstorage.Config{Bucket: "bucket", Prefix: "deuterium-test/"}, S3: sdk}}
	ctx := context.Background()
	for n := 0; n < 2; n++ {
		if e := objects.Retain(ctx, "deuterium-test/chat-media/media_1/image.webp", target, `"digest"`, 100); e != nil {
			t.Fatal(e)
		}
	}
	if copies != 1 {
		t.Fatal("restart copied twice", copies)
	}
	for _, key := range []string{"deuterium-test/chat-media/media_1/image.webp", "deuterium-test/image-review/evidence/../image.webp", "other/image-review/evidence/" + id + ".image"} {
		if e := objects.DeleteEvidence(ctx, key); !errors.Is(e, ErrInvalidImage) {
			t.Fatal("source delete allowed", key, e)
		}
	}
	if deletes != 0 {
		t.Fatal("unsafe request reached OSS")
	}
	if e := objects.DeleteEvidence(ctx, target); e != nil {
		t.Fatal(e)
	}
	if e := objects.DeleteEvidence(ctx, target); e != nil {
		t.Fatal("delete not idempotent", e)
	}
}
