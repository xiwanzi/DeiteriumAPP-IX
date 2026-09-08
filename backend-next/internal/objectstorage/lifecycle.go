package objectstorage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

func missingObject(err error) bool {
	var api smithy.APIError
	return errors.As(err, &api) && (api.ErrorCode() == "NotFound" || api.ErrorCode() == "NoSuchKey" || api.ErrorCode() == "404")
}

func (c *Client) safeObjectKey(key string) bool {
	return strings.HasPrefix(key, c.Config.Prefix) && len(key) > len(c.Config.Prefix) && !strings.Contains(key, "..") && !strings.ContainsAny(key, "\\\r\n")
}

// CopyVerified is restartable: an existing correct target is reused without resetting its age.
// No original object is deleted here. The caller must first persist the verified target location.
func (c *Client) CopyVerified(ctx context.Context, source, target, expectedSHA string) (time.Time, error) {
	if !c.safeObjectKey(source) || !c.safeObjectKey(target) || source == target {
		return time.Time{}, ErrUnavailable
	}
	head, err := c.S3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(source)})
	if err != nil {
		if missingObject(err) && expectedSHA != "" {
			copyHead, lookupErr := c.S3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(target)})
			if lookupErr == nil {
				return c.verifyCopyDigest(ctx, target, copyHead, expectedSHA)
			}
		}
		return time.Time{}, ErrUnavailable
	}
	if aws.ToInt64(head.ContentLength) < 1 || aws.ToInt64(head.ContentLength) > 20<<20 {
		return time.Time{}, ErrInvalidObject
	}
	copyHead, err := c.S3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(target)})
	if err != nil && !missingObject(err) {
		return time.Time{}, ErrUnavailable
	}
	if missingObject(err) {
		_, err = c.S3.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket: aws.String(c.Config.Bucket), Key: aws.String(target),
			CopySource: aws.String((&url.URL{Path: c.Config.Bucket + "/" + source}).EscapedPath()), CopySourceIfMatch: head.ETag,
		})
		if err != nil {
			return time.Time{}, ErrUnavailable
		}
		copyHead, err = c.S3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(target)})
		if err != nil {
			return time.Time{}, ErrUnavailable
		}
	}
	if aws.ToInt64(copyHead.ContentLength) != aws.ToInt64(head.ContentLength) || aws.ToString(copyHead.ContentType) != aws.ToString(head.ContentType) || copyHead.LastModified == nil {
		return time.Time{}, ErrInvalidObject
	}
	if expectedSHA == "" && (aws.ToString(head.ETag) == "" || aws.ToString(head.ETag) != aws.ToString(copyHead.ETag)) {
		return time.Time{}, ErrInvalidObject
	}
	return c.verifyCopyDigest(ctx, target, copyHead, expectedSHA)
}

func (c *Client) verifyCopyDigest(ctx context.Context, target string, copyHead *s3.HeadObjectOutput, expectedSHA string) (time.Time, error) {
	if copyHead.LastModified == nil || aws.ToInt64(copyHead.ContentLength) < 1 || aws.ToInt64(copyHead.ContentLength) > 20<<20 {
		return time.Time{}, ErrInvalidObject
	}
	if expectedSHA != "" {
		object, err := c.S3.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(target), IfMatch: copyHead.ETag})
		if err != nil {
			return time.Time{}, ErrUnavailable
		}
		digest := sha256.New()
		n, readErr := io.Copy(digest, io.LimitReader(object.Body, aws.ToInt64(copyHead.ContentLength)+1))
		object.Body.Close()
		if readErr != nil || n != aws.ToInt64(copyHead.ContentLength) || hex.EncodeToString(digest.Sum(nil)) != expectedSHA {
			return time.Time{}, ErrInvalidObject
		}
	}
	return copyHead.LastModified.UTC(), nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	if !c.safeObjectKey(key) {
		return ErrUnavailable
	}
	_, err := c.S3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(key)})
	if err != nil && !missingObject(err) {
		return ErrUnavailable
	}
	return nil
}

func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	if !c.safeObjectKey(key) {
		return false, ErrUnavailable
	}
	_, err := c.S3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(key)})
	if missingObject(err) {
		return false, nil
	}
	if err != nil {
		return false, ErrUnavailable
	}
	return true, nil
}
