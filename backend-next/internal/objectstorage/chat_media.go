package objectstorage

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"image"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ChatImage struct {
	VerifiedImage
	Animated   bool
	Frames     int
	DurationMS int
}

func (c *Client) ReadChatObject(ctx context.Context, key string, limit int64) ([]byte, error) {
	if !c.safeObjectKey(key) || limit < 1 || limit > 2<<20 {
		return nil, ErrInvalidObject
	}
	o, err := c.S3.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(key)})
	if err != nil {
		return nil, ErrUnavailable
	}
	defer o.Body.Close()
	if aws.ToInt64(o.ContentLength) != limit {
		return nil, ErrInvalidObject
	}
	data, err := io.ReadAll(io.LimitReader(o.Body, limit+1))
	if err != nil {
		return nil, ErrUnavailable
	}
	if int64(len(data)) != limit {
		return nil, ErrInvalidObject
	}
	return data, nil
}

func (c *Client) DownloadUntil(ctx context.Context, key string, deadline time.Time) (string, time.Time, error) {
	if !c.safeObjectKey(key) {
		return "", time.Time{}, ErrUnavailable
	}
	duration := min(time.Until(deadline), 15*time.Minute)
	if duration < time.Second {
		return "", time.Time{}, ErrUnavailable
	}
	r, err := c.Presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(c.Config.Bucket), Key: aws.String(key)}, func(o *s3.PresignOptions) { o.Expires = duration })
	if err != nil {
		return "", time.Time{}, ErrUnavailable
	}
	return r.URL, time.Now().UTC().Add(duration), nil
}

type webpChunk struct {
	kind string
	data []byte
}

func webpChunks(data []byte) ([]webpChunk, error) {
	out := []webpChunk{}
	for pos := 0; pos < len(data); {
		if len(data)-pos < 8 {
			return nil, ErrInvalidObject
		}
		n := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		end := pos + 8 + n
		if n < 0 || end > len(data) || end+(n&1) > len(data) {
			return nil, ErrInvalidObject
		}
		out = append(out, webpChunk{string(data[pos : pos+4]), data[pos+8 : end]})
		pos = end + (n & 1)
		if len(out) > 1024 {
			return nil, ErrInvalidObject
		}
	}
	return out, nil
}
func uint24(b []byte) int { return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 }
func appendChunk(out []byte, kind string, data []byte) []byte {
	out = append(out, kind...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(data)))
	out = append(out, data...)
	if len(data)%2 != 0 {
		out = append(out, 0)
	}
	return out
}
func webpFrame(data []byte, w, h int) []byte {
	x := make([]byte, 10)
	if len(data) >= 4 && string(data[:4]) == "ALPH" {
		x[0] = 0x10
	}
	for i := 0; i < 3; i++ {
		x[4+i] = byte((w - 1) >> (8 * i))
		x[7+i] = byte((h - 1) >> (8 * i))
	}
	body := appendChunk([]byte("WEBP"), "VP8X", x)
	body = append(body, data...)
	out := binary.LittleEndian.AppendUint32([]byte("RIFF"), uint32(len(body)))
	return append(out, body...)
}

// Validate every frame with bounded dimensions before allocating its pixel
// buffer. No encoding or pixel transformations run on the server.
func VerifyChatImage(data []byte, expectedSHA, expectedMD5 string, thumbnail bool) (ChatImage, error) {
	var result ChatImage
	maxBytes := 2 << 20
	if thumbnail {
		maxBytes = 80 << 10
	}
	if len(data) < 20 || len(data) > maxBytes || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" || int(binary.LittleEndian.Uint32(data[4:8])) != len(data)-8 {
		return result, ErrInvalidObject
	}
	h := sha256.Sum256(data)
	d := md5.Sum(data)
	result.SHA256 = hex.EncodeToString(h[:])
	result.MD5 = base64.StdEncoding.EncodeToString(d[:])
	if result.SHA256 != expectedSHA || result.MD5 != expectedMD5 {
		return result, ErrInvalidObject
	}
	chunks, err := webpChunks(data[12:])
	if err != nil {
		return result, err
	}
	extended := false
	for _, ch := range chunks {
		if ch.kind == "VP8X" {
			if len(ch.data) != 10 || extended {
				return result, ErrInvalidObject
			}
			extended = true
			result.Animated = ch.data[0]&2 != 0
			result.Width = 1 + uint24(ch.data[4:7])
			result.Height = 1 + uint24(ch.data[7:10])
		}
	}
	if !result.Animated {
		for _, ch := range chunks {
			if ch.kind == "ANIM" || ch.kind == "ANMF" {
				return result, ErrInvalidObject
			}
		}
		cfg, format, e := image.DecodeConfig(bytes.NewReader(data))
		if e != nil || format != "webp" || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 8192 || cfg.Height > 8192 || int64(cfg.Width)*int64(cfg.Height) > 8_000_000 {
			return result, ErrInvalidObject
		}
		if thumbnail && (cfg.Width > 320 || cfg.Height > 320) {
			return result, ErrInvalidObject
		}
		if _, _, e = image.Decode(bytes.NewReader(data)); e != nil {
			return result, ErrInvalidObject
		}
		result.Width = cfg.Width
		result.Height = cfg.Height
		result.Frames = 1
		return result, nil
	}
	if thumbnail || result.Width < 1 || result.Height < 1 || result.Width > 512 || result.Height > 512 {
		return result, ErrInvalidObject
	}
	anim := false
	var pixels int64
	for _, ch := range chunks {
		if ch.kind == "ANIM" {
			if anim || len(ch.data) != 6 {
				return result, ErrInvalidObject
			}
			anim = true
		}
		if ch.kind != "ANMF" {
			continue
		}
		if !anim || len(ch.data) < 24 {
			return result, ErrInvalidObject
		}
		x, y := 2*uint24(ch.data[:3]), 2*uint24(ch.data[3:6])
		w, h := 1+uint24(ch.data[6:9]), 1+uint24(ch.data[9:12])
		duration := uint24(ch.data[12:15])
		result.Frames++
		result.DurationMS += duration
		pixels += int64(w) * int64(h)
		if x+w > result.Width || y+h > result.Height || ch.data[15]&0xfc != 0 || duration < 10 || result.Frames > 300 || result.DurationMS > 30_000 || pixels > 80_000_000 {
			return result, ErrInvalidObject
		}
		frame := webpFrame(ch.data[16:], w, h)
		subchunks, e := webpChunks(ch.data[16:])
		if e != nil {
			return result, e
		}
		bitstreams := 0
		for _, sub := range subchunks {
			if sub.kind == "VP8 " || sub.kind == "VP8L" {
				bitstreams++
			} else if sub.kind != "ALPH" {
				return result, ErrInvalidObject
			}
		}
		if bitstreams != 1 {
			return result, ErrInvalidObject
		}
		cfg, format, e := image.DecodeConfig(bytes.NewReader(frame))
		if e != nil || format != "webp" || cfg.Width != w || cfg.Height != h {
			return result, ErrInvalidObject
		}
		if _, _, e = image.Decode(bytes.NewReader(frame)); e != nil {
			return result, ErrInvalidObject
		}
	}
	if result.Frames < 1 {
		return result, ErrInvalidObject
	}
	return result, nil
}
