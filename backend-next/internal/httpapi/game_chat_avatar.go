package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var avatarHTTP = &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
var textureHash = regexp.MustCompile(`^/texture/[a-fA-F0-9]{32,64}$`)
var avatarUUID = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)

func avatarGet(ctx context.Context, address string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", address, nil)
	if err != nil {
		return nil, err
	}
	r, err := avatarHTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("skin upstream status %d", r.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("skin response too large")
	}
	return data, nil
}
func gameAvatarTextureURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Host != "textures.minecraft.net" || u.RawQuery != "" || u.Fragment != "" || !textureHash.MatchString(u.Path) || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("invalid skin texture URL")
	}
	u.Scheme = "https"
	return u.String(), nil
}
func fetchGameAvatar(ctx context.Context, uuid string) ([]byte, error) {
	compact := strings.ReplaceAll(uuid, "-", "")
	if !avatarUUID.MatchString(compact) {
		return nil, fmt.Errorf("invalid game UUID")
	}
	data, err := avatarGet(ctx, "https://sessionserver.mojang.com/session/minecraft/profile/"+compact, 65536)
	if err != nil {
		return nil, err
	}
	var profile struct {
		ID         string `json:"id"`
		Properties []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"properties"`
	}
	if err = json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}
	if !strings.EqualFold(profile.ID, compact) {
		return nil, fmt.Errorf("skin profile mismatch")
	}
	for _, p := range profile.Properties {
		if p.Name == "textures" {
			raw, e := base64.StdEncoding.DecodeString(p.Value)
			if e != nil {
				return nil, e
			}
			var textures struct {
				Textures struct {
					Skin struct {
						URL string `json:"url"`
					} `json:"SKIN"`
				} `json:"textures"`
			}
			if e = json.Unmarshal(raw, &textures); e != nil {
				return nil, e
			}
			address, e := gameAvatarTextureURL(textures.Textures.Skin.URL)
			if e != nil {
				return nil, e
			}
			skin, e := avatarGet(ctx, address, 1<<20)
			if e != nil {
				return nil, e
			}
			return composeGameAvatar(skin)
		}
	}
	return nil, fmt.Errorf("profile has no custom skin")
}
func composeGameAvatar(skin []byte) ([]byte, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(skin))
	if err != nil {
		return nil, err
	}
	if cfg.Width != 64 || (cfg.Height != 64 && cfg.Height != 32) {
		return nil, fmt.Errorf("unsupported skin dimensions")
	}
	src, err := png.Decode(bytes.NewReader(skin))
	if err != nil {
		return nil, err
	}
	face := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	draw.Draw(face, face.Bounds(), src, image.Pt(8, 8), draw.Src)
	draw.Draw(face, face.Bounds(), src, image.Pt(40, 8), draw.Over)
	var result bytes.Buffer
	err = png.Encode(&result, face)
	return result.Bytes(), err
}
func (s *Server) gameAvatar(w http.ResponseWriter, r *http.Request) {
	if _, err := s.authenticate(r); err != nil {
		failError(w, r, err)
		return
	}
	fetch := s.GameAvatarResolver
	if fetch == nil {
		fetch = fetchGameAvatar
	}
	data, err := s.Store.GameChatAvatar(r.Context(), r.PathValue("gameId"), fetch)
	if err != nil {
		socialFailureV2(w, r, err)
		return
	}
	success(w, r, map[string]any{"png": base64.StdEncoding.EncodeToString(data)})
}
