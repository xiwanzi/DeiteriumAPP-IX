package objectstorage

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	_ "golang.org/x/image/webp"
)

var ErrUnavailable = errors.New("object storage unavailable")
var ErrInvalidObject = errors.New("uploaded image failed verification")

type Config struct{ Endpoint, Region, Bucket, Prefix, AccessKey, SecretKey string }
type Client struct {
	Config  Config
	S3      *s3.Client
	Presign *s3.PresignClient
}
type Authorization struct {
	Provider      string            `json:"provider"`
	Method        string            `json:"method"`
	UploadURL     string            `json:"uploadUrl"`
	SignedHeaders map[string]string `json:"signedHeaders"`
	ExpiresAt     time.Time         `json:"expiresAt"`
	SizeBytes     int64             `json:"sizeBytes"`
}
type VerifiedImage struct {
	Width, Height int
	SHA256, MD5   string
}

func FromEnvironment() (*Client, error) {
	// Configuration failures must not poison later callers (or another server
	// instance in the same process). SDK clients share the HTTP transport.
	return New(Config{Endpoint: os.Getenv("DEUTERIUM_S3_ENDPOINT"), Region: os.Getenv("DEUTERIUM_S3_REGION"), Bucket: os.Getenv("DEUTERIUM_S3_BUCKET"), Prefix: os.Getenv("DEUTERIUM_S3_PREFIX"), AccessKey: os.Getenv("DEUTERIUM_S3_ACCESS_KEY"), SecretKey: os.Getenv("DEUTERIUM_S3_SECRET_KEY")})
}

func New(c Config) (*Client, error) {
	u, err := url.Parse(c.Endpoint)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" || c.Bucket == "" || c.Region == "" || c.AccessKey == "" || c.SecretKey == "" {
		return nil, ErrUnavailable
	}
	if c.Prefix == "" {
		c.Prefix = "deuterium-test/"
	}
	if strings.Contains(c.Prefix, "..") || strings.HasPrefix(c.Prefix, "/") || !strings.HasSuffix(c.Prefix, "/") {
		return nil, ErrUnavailable
	}
	sdk := s3.NewFromConfig(aws.Config{Region: c.Region, Credentials: credentials.NewStaticCredentialsProvider(c.AccessKey, c.SecretKey, ""), HTTPClient: &http.Client{Timeout: 30 * time.Second}}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(c.Endpoint)
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})
	return &Client{Config: c, S3: sdk, Presign: s3.NewPresignClient(sdk)}, nil
}

func (c *Client) Upload(ctx context.Context, key, contentType, md5Value string, size int64, expires time.Time) (*Authorization, error) {
	if !strings.HasPrefix(key, c.Config.Prefix) {
		return nil, ErrUnavailable
	}
	duration := time.Until(expires)
	if duration > 5*time.Minute {
		duration = 5 * time.Minute
	}
	if duration <= 0 {
		return nil, ErrUnavailable
	}
	request, err := c.Presign.PresignPutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(key), ContentType: aws.String(contentType), ContentLength: aws.Int64(size), ContentMD5: aws.String(md5Value), IfNoneMatch: aws.String("*")}, func(o *s3.PresignOptions) { o.Expires = duration })
	if err != nil {
		return nil, ErrUnavailable
	}
	headers := map[string]string{}
	for name, values := range request.SignedHeader {
		if !strings.EqualFold(name, "Host") && !strings.EqualFold(name, "Content-Length") {
			headers[textproto.CanonicalMIMEHeaderKey(name)] = strings.Join(values, ",")
		}
	}
	headers["Content-Type"] = contentType
	headers["Content-Md5"] = md5Value
	headers["If-None-Match"] = "*"
	return &Authorization{Provider: "RAINYUN_S3", Method: request.Method, UploadURL: request.URL, SignedHeaders: headers, ExpiresAt: time.Now().UTC().Add(duration), SizeBytes: size}, nil
}

func (c *Client) Download(ctx context.Context, key string) (string, time.Time, error) {
	if !strings.HasPrefix(key, c.Config.Prefix) {
		return "", time.Time{}, ErrUnavailable
	}
	expires := time.Now().UTC().Add(15 * time.Minute)
	r, err := c.Presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(key)}, func(o *s3.PresignOptions) { o.Expires = 15 * time.Minute })
	if err != nil {
		return "", time.Time{}, ErrUnavailable
	}
	return r.URL, expires, nil
}

func VerifyImage(data []byte, contentType, md5Value string, size int64) (VerifiedImage, error) {
	var verified VerifiedImage
	if int64(len(data)) != size || size < 1 || size > 20<<20 {
		return verified, ErrInvalidObject
	}
	md5sum := md5.Sum(data)
	verified.MD5 = base64.StdEncoding.EncodeToString(md5sum[:])
	if verified.MD5 != md5Value {
		return verified, ErrInvalidObject
	}
	c, format, err := image.DecodeConfig(bytes.NewReader(data))
	expected := map[string]string{"image/png": "png", "image/jpeg": "jpeg", "image/webp": "webp"}[contentType]
	if err != nil || expected == "" || format != expected || c.Width < 1 || c.Height < 1 || c.Width > 8192 || c.Height > 8192 || int64(c.Width)*int64(c.Height) > 16_000_000 {
		return verified, ErrInvalidObject
	}
	decoded, actualFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil || actualFormat != format || decoded.Bounds().Dx() != c.Width || decoded.Bounds().Dy() != c.Height {
		return verified, ErrInvalidObject
	}
	verified.Width, verified.Height = c.Width, c.Height
	sha := sha256.Sum256(data)
	verified.SHA256 = hex.EncodeToString(sha[:])
	return verified, nil
}

func (c *Client) Verify(ctx context.Context, key, contentType, md5Value string, size int64) (VerifiedImage, error) {
	var empty VerifiedImage
	if !strings.HasPrefix(key, c.Config.Prefix) {
		return empty, ErrUnavailable
	}
	head, err := c.S3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(key)})
	if err != nil {
		return empty, ErrUnavailable
	}
	if aws.ToInt64(head.ContentLength) != size {
		return empty, ErrInvalidObject
	}
	object, err := c.S3.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(key), IfMatch: head.ETag})
	if err != nil {
		return empty, ErrUnavailable
	}
	defer object.Body.Close()
	data, err := io.ReadAll(io.LimitReader(object.Body, size+1))
	if err != nil {
		return empty, ErrUnavailable
	}
	return VerifyImage(data, contentType, md5Value, size)
}
