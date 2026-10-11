package httpapi

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestGameAvatarComposesBaseAndTransparentOuterLayer(t *testing.T) {
	skin := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	skin.SetNRGBA(8, 8, color.NRGBA{R: 200, A: 255})
	skin.SetNRGBA(9, 8, color.NRGBA{B: 200, A: 255})
	skin.SetNRGBA(41, 8, color.NRGBA{G: 200, A: 255})
	var raw bytes.Buffer
	_ = png.Encode(&raw, skin)
	out, err := composeGameAvatar(raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds() != image.Rect(0, 0, 8, 8) || len(out) > 4096 {
		t.Fatal("unexpected avatar size")
	}
	if color.NRGBAModel.Convert(decoded.At(0, 0)) != (color.NRGBA{R: 200, A: 255}) || color.NRGBAModel.Convert(decoded.At(1, 0)) != (color.NRGBA{G: 200, A: 255}) {
		t.Fatal("base or outer skin layer lost")
	}
}
func TestGameAvatarTextureURLIsPinned(t *testing.T) {
	valid := "http://textures.minecraft.net/texture/" + strings.Repeat("a", 64)
	if got, err := gameAvatarTextureURL(valid); err != nil || !strings.HasPrefix(got, "https:") {
		t.Fatal(got, err)
	}
	for _, bad := range []string{"https://127.0.0.1/texture/" + strings.Repeat("a", 64), valid + "?redirect=1", strings.Replace(valid, "textures.minecraft.net", "textures.minecraft.net:443", 1), strings.Replace(valid, "textures.minecraft.net", "evil@textures.minecraft.net", 1)} {
		if _, err := gameAvatarTextureURL(bad); err == nil {
			t.Fatal("unsafe texture accepted", bad)
		}
	}
}
