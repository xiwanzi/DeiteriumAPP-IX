package store

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
)

type commerceCursorV2 struct {
	Scope    string `json:"scope"`
	Sequence int64  `json:"sequence"`
	Amount   string `json:"amount"`
	Sort     string `json:"sort"`
}

func CommerceCursorV2(scope string, record CommerceRecordV2, sort string) string {
	if sort == "" {
		sort = "NEWEST"
	}
	amount := ""
	if sort != "NEWEST" {
		amount = record.Amount
	}
	raw, _ := json.Marshal(commerceCursorV2{Digest([]byte(scope))[:16], record.Sequence, amount, sort})
	return base64.RawURLEncoding.EncodeToString(raw)
}
func CommerceParseCursorV2(scope, cursor, sort string) (int64, string, error) {
	if cursor == "" {
		return 0, "", nil
	}
	if sort == "" {
		sort = "NEWEST"
	}
	if len(cursor) > 512 {
		return 0, "", catalogInvalid()
	}
	raw, e := base64.RawURLEncoding.DecodeString(cursor)
	if e != nil {
		return 0, "", catalogInvalid()
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	var value commerceCursorV2
	if d.Decode(&value) != nil {
		return 0, "", catalogInvalid()
	}
	var extra any
	if d.Decode(&extra) != io.EOF || value.Scope != Digest([]byte(scope))[:16] || value.Sort != sort || value.Sequence < 1 {
		return 0, "", catalogInvalid()
	}
	if sort == "NEWEST" {
		if value.Amount != "" {
			return 0, "", catalogInvalid()
		}
	} else if sort == "REWARD_DESC" || sort == "REWARD_ASC" {
		n, e := commerceAmountV2(value.Amount)
		if e != nil || n.Sign() <= 0 {
			return 0, "", catalogInvalid()
		}
	} else {
		return 0, "", catalogInvalid()
	}
	return value.Sequence, value.Amount, nil
}
