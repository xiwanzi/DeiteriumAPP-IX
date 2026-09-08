package bridge

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/config"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/identity"
	"github.com/xiwanzi/DeuteriumAPP/backend-next/internal/store"
)

var ErrProtocol = errors.New("invalid protocol message")
var messageID = regexp.MustCompile(`^[A-Za-z0-9_:.\-]{1,128}$`)
var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var itemRefPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}:[a-z0-9_./-]{1,63}$`)

type Envelope struct {
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	EventID   string          `json:"eventId,omitempty"`
	SentAt    string          `json:"sentAt,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

func Decode(data []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(target) != nil {
		return ErrProtocol
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return ErrProtocol
	}
	return nil
}

func ValidMessageID(s string) bool { return messageID.MatchString(s) }

func Content(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > 256 || !utf8.ValidString(s) {
		return "", ErrProtocol
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return "", ErrProtocol
		}
	}
	return s, nil
}

func ValidateCore(c config.Config, node config.Node, e Envelope) (payload []byte, chat *store.CoreChat, item *store.ItemVersion, err error) {
	if !ValidMessageID(e.EventID) || len(e.Type) > 64 {
		return nil, nil, nil, ErrProtocol
	}
	switch e.Type {
	case "chat.public.event":
		if !node.Chat {
			return nil, nil, nil, ErrProtocol
		}
		chat = &store.CoreChat{}
		if Decode(e.Payload, chat) != nil || !identity.ValidUUID(chat.PlayerUUID) || !identity.ValidGameID(chat.GameID) {
			return nil, nil, nil, ErrProtocol
		}
		chat.Content, err = Content(chat.Content)
		if err != nil {
			return
		}
		payload, err = json.Marshal(chat)
	case "item.version.published":
		item = &store.ItemVersion{}
		if node.ItemPrefix == "" || Decode(e.Payload, item) != nil || !strings.HasPrefix(item.ItemRef, node.ItemPrefix) || !itemRefPattern.MatchString(item.ItemRef) || len(item.ItemRef) > 96 || item.Revision < 1 || item.Revision > 2147483647 || !sha256Pattern.MatchString(item.PayloadSHA256) || len(item.DisplayName) == 0 || utf8.RuneCountInString(item.DisplayName) > 128 || utf8.RuneCountInString(item.Description) > 1024 || item.MaxQuantity < 1 || item.MaxQuantity > 1000000 || len(item.CompatibleServerIDs) == 0 || len(item.CompatibleServerIDs) > 32 || len(item.RequiredMods) > 128 {
			return nil, nil, nil, ErrProtocol
		}
		seen := map[string]bool{}
		for _, id := range item.CompatibleServerIDs {
			if seen[id] || !c.HasNode(id) {
				return nil, nil, nil, ErrProtocol
			}
			seen[id] = true
		}
		for _, mod := range item.RequiredMods {
			if !config.NodeID.MatchString(mod) {
				return nil, nil, nil, ErrProtocol
			}
		}
		sort.Strings(item.CompatibleServerIDs)
		sort.Strings(item.RequiredMods)
		payload, err = json.Marshal(item)
	default:
		err = ErrProtocol
	}
	return
}
