package imagereview

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/objectstorage"
)

type ObjectInfo struct {
	ETag        string
	Size        int64
	ContentType string
}
type Objects interface {
	Inspect(context.Context, Candidate) (ObjectInfo, error)
	URL(context.Context, string) (string, error)
	EvidenceKey(string) string
	Retain(context.Context, string, string, string, int64) error
	DeleteEvidence(context.Context, string) error
}
type S3Objects struct{ Client *objectstorage.Client }

func (o *S3Objects) safeSource(key string) bool {
	p := o.Client.Config.Prefix
	return strings.HasPrefix(key, p) && len(key) > len(p) && !strings.Contains(key, "..") && !strings.ContainsAny(key, "\\\r\n") && !strings.HasPrefix(key, p+"image-review/")
}
func (o *S3Objects) EvidenceKey(id string) string {
	if !ValidID(id) {
		return ""
	}
	return o.Client.Config.Prefix + "image-review/evidence/" + id + ".image"
}
func (o *S3Objects) safeEvidence(key string) bool {
	prefix := o.Client.Config.Prefix + "image-review/evidence/"
	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, ".image") {
		return false
	}
	return ValidID(strings.TrimSuffix(strings.TrimPrefix(key, prefix), ".image"))
}
func storageError(e error) error {
	var a smithy.APIError
	if errors.As(e, &a) {
		switch a.ErrorCode() {
		case "NoSuchKey", "NotFound", "404":
			return ErrObjectMissing
		}
	}
	return &ReviewError{Code: "OBJECT_STORAGE_UNAVAILABLE"}
}
func (o *S3Objects) Inspect(ctx context.Context, c Candidate) (ObjectInfo, error) {
	var info ObjectInfo
	if !o.safeSource(c.SourceKey) {
		return info, ErrInvalidImage
	}
	head, e := o.Client.S3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(o.Client.Config.Bucket), Key: aws.String(c.SourceKey)})
	if e != nil {
		return info, storageError(e)
	}
	info = ObjectInfo{ETag: aws.ToString(head.ETag), Size: aws.ToInt64(head.ContentLength), ContentType: c.ContentType}
	if info.ETag == "" || info.Size < 1 || info.Size > 20<<20 || info.Size != c.SizeBytes {
		return info, ErrInvalidImage
	}
	// A bounded in-memory header read detects animation even for a staged upload.
	object, e := o.Client.S3.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(o.Client.Config.Bucket), Key: aws.String(c.SourceKey), Range: aws.String("bytes=0-65535"), IfMatch: head.ETag})
	if e != nil {
		return info, storageError(e)
	}
	defer object.Body.Close()
	data, e := io.ReadAll(io.LimitReader(object.Body, 65537))
	if e != nil || len(data) > 65536 {
		return info, ErrInvalidImage
	}
	mime, animated, e := inspectHeader(data)
	if e != nil {
		return info, e
	}
	if c.Animated || animated {
		return info, ErrAnimated
	}
	if mime != c.ContentType {
		return info, ErrInvalidImage
	}
	return info, nil
}
func inspectHeader(data []byte) (string, bool, error) {
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		if len(data) < 21 {
			return "", false, ErrInvalidImage
		}
		switch string(data[12:16]) {
		case "VP8X":
			return "image/webp", data[20]&2 != 0, nil
		case "VP8 ", "VP8L":
			return "image/webp", false, nil
		}
		return "", false, ErrInvalidImage
	}
	if len(data) >= 6 && (string(data[:6]) == "GIF89a" || string(data[:6]) == "GIF87a") {
		return "image/gif", true, nil
	}
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return "image/jpeg", false, nil
	}
	if bytes.HasPrefix(data, []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		for at := 8; at+12 <= len(data); {
			n := uint64(binary.BigEndian.Uint32(data[at : at+4]))
			kind := string(data[at+4 : at+8])
			if kind == "acTL" {
				return "image/png", true, nil
			}
			if kind == "IDAT" {
				return "image/png", false, nil
			}
			if n > 65536 || uint64(at)+n+12 > uint64(len(data)) {
				break
			}
			at += int(n) + 12
		}
	}
	return "", false, ErrInvalidImage
}
func (o *S3Objects) URL(ctx context.Context, key string) (string, error) {
	if !o.safeSource(key) && !o.safeEvidence(key) {
		return "", ErrInvalidImage
	}
	u, _, e := o.Client.Download(ctx, key)
	return u, e
}
func (o *S3Objects) Retain(ctx context.Context, source, target, etag string, size int64) error {
	if !o.safeSource(source) || !o.safeEvidence(target) || etag == "" || size < 1 || size > 20<<20 {
		return ErrInvalidImage
	}
	// A known destination is persisted before the copy; retries verify/reuse it.
	targetHead, e := o.Client.S3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(o.Client.Config.Bucket), Key: aws.String(target)})
	if e == nil {
		if aws.ToInt64(targetHead.ContentLength) == size && aws.ToString(targetHead.ETag) == etag {
			return nil
		}
		return &ReviewError{Code: "EVIDENCE_INTEGRITY_FAILED"}
	}
	if !errors.Is(storageError(e), ErrObjectMissing) {
		return storageError(e)
	}
	copySource := o.Client.Config.Bucket + "/" + strings.ReplaceAll(url.PathEscape(source), "%2F", "/")
	_, e = o.Client.S3.CopyObject(ctx, &s3.CopyObjectInput{Bucket: aws.String(o.Client.Config.Bucket), Key: aws.String(target), CopySource: aws.String(copySource), CopySourceIfMatch: aws.String(etag)})
	if e != nil {
		return storageError(e)
	}
	head, e := o.Client.S3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(o.Client.Config.Bucket), Key: aws.String(target)})
	if e != nil {
		return storageError(e)
	}
	if aws.ToInt64(head.ContentLength) != size || aws.ToString(head.ETag) != etag {
		return &ReviewError{Code: "EVIDENCE_INTEGRITY_FAILED"}
	}
	return nil
}
func (o *S3Objects) DeleteEvidence(ctx context.Context, key string) error {
	if !o.safeEvidence(key) {
		return ErrInvalidImage
	}
	_, e := o.Client.S3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(o.Client.Config.Bucket), Key: aws.String(key)})
	if e != nil {
		return storageError(e)
	}
	// Do not declare space released until a subsequent read sees the deletion.
	_, e = o.Client.S3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(o.Client.Config.Bucket), Key: aws.String(key)})
	if e != nil && errors.Is(storageError(e), ErrObjectMissing) {
		return nil
	}
	return &ReviewError{Code: "EVIDENCE_DELETE_PENDING"}
}
