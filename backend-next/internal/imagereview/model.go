// Package imagereview is an independent, asynchronous observer of uploaded images.
// It never changes an upload, a message, a player, or a source object.
package imagereview

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const PolicyVersion = "static-v1-20261012"
const Model = "deepseek-flash"
const Threshold = 50

var ErrNotFound = errors.New("review not found")
var ErrObjectMissing = errors.New("image missing")
var ErrInvalidImage = errors.New("unsupported image")
var ErrAnimated = errors.New("animated image skipped")

type Scores struct {
	R18       int `json:"r18"`
	Politics  int `json:"politics"`
	Promotion int `json:"promotion"`
}

func (s Scores) Total() int {
	high := max(s.R18, s.Politics, s.Promotion)
	return min(100, high+(s.R18+s.Politics+s.Promotion-high)/4)
}

type Result struct {
	Scores  Scores   `json:"scores"`
	Reasons []string `json:"reasons"`
	Summary string   `json:"summary"`
}

func (r Result) Validate() error {
	for _, n := range []int{r.Scores.R18, r.Scores.Politics, r.Scores.Promotion} {
		if n < 0 || n > 100 {
			return errors.New("invalid model score")
		}
	}
	if strings.TrimSpace(r.Summary) == "" || len([]rune(r.Summary)) > 300 || len(r.Reasons) < 1 || len(r.Reasons) > 6 {
		return errors.New("invalid model explanation")
	}
	for _, s := range r.Reasons {
		if strings.TrimSpace(s) == "" || len([]rune(s)) > 300 {
			return errors.New("invalid model explanation")
		}
	}
	return nil
}

type Candidate struct {
	ID          string    `json:"id"`
	SourceKind  string    `json:"sourceKind"`
	SourceID    string    `json:"sourceId"`
	MediaID     string    `json:"mediaId,omitempty"`
	SourceKey   string    `json:"-"`
	SHA256      string    `json:"-"`
	Verified    bool      `json:"-"`
	ContentType string    `json:"-"`
	SizeBytes   int64     `json:"sizeBytes"`
	Animated    bool      `json:"-"`
	UploaderID  string    `json:"uploaderId"`
	PlayerRef   string    `json:"playerRef"`
	GameID      string    `json:"gameId"`
	DisplayName string    `json:"displayName"`
	Purpose     string    `json:"purpose"`
	UploadedAt  time.Time `json:"uploadedAt"`
	ExpiresAt   time.Time `json:"-"`
	SourceState string    `json:"-"`
}

func ReviewID(kind, id string) string {
	sum := sha256.Sum256([]byte(kind + ":" + id))
	return hex.EncodeToString(sum[:])
}
func ValidID(id string) bool {
	b, e := hex.DecodeString(id)
	return e == nil && len(b) == 32 && id == strings.ToLower(id)
}

type Item struct {
	Candidate
	State           string     `json:"state"`
	ThresholdUsed   int        `json:"thresholdUsed"`
	Score           *int       `json:"score"`
	Result          *Result    `json:"result,omitempty"`
	PolicyVersion   string     `json:"policyVersion"`
	Model           string     `json:"model"`
	EvidenceKey     string     `json:"-"`
	EvidenceETag    string     `json:"-"`
	ErrorCode       string     `json:"errorCode,omitempty"`
	Attempts        int        `json:"attempts"`
	ReviewedAt      *time.Time `json:"reviewedAt,omitempty"`
	NextAttemptAt   time.Time  `json:"-"`
	Events          []Event    `json:"events,omitempty"`
	EventsTruncated bool       `json:"eventsTruncated,omitempty"`
}
type Event struct {
	ID          string    `json:"id"`
	Channel     string    `json:"channel"`
	SenderID    string    `json:"senderId"`
	PlayerRef   string    `json:"playerRef"`
	GameID      string    `json:"gameId"`
	DisplayName string    `json:"displayName"`
	SentAt      time.Time `json:"sentAt"`
}
type Summary struct {
	Flagged     int        `json:"flagged"`
	Clear       int        `json:"clear"`
	Pending     int        `json:"pending"`
	Retrying    int        `json:"retrying"`
	Skipped     int        `json:"skipped"`
	Unavailable int        `json:"unavailable"`
	Deleting    int        `json:"deleting"`
	LastScanAt  *time.Time `json:"lastScanAt"`
	LastWorkAt  *time.Time `json:"lastWorkAt"`
	ScanError   string     `json:"scanError,omitempty"`
	Model       string     `json:"model"`
	ModelLabel  string     `json:"modelLabel"`
	Threshold   int        `json:"threshold"`
}
type ReviewError struct {
	Code  string
	Delay time.Duration
}

func (e *ReviewError) Error() string { return e.Code }
func errorCode(err error) string {
	var re *ReviewError
	if errors.As(err, &re) {
		return re.Code
	}
	if errors.Is(err, ErrObjectMissing) {
		return "SOURCE_NOT_FOUND"
	}
	if errors.Is(err, ErrInvalidImage) {
		return "INVALID_IMAGE"
	}
	return "REVIEW_TEMPORARY_FAILURE"
}
func resultJSON(r Result) string { b, _ := json.Marshal(r); return string(b) }
func sourceKey(prefix, media string) string {
	return fmt.Sprintf("%schat-media/%s/image.webp", prefix, media)
}
