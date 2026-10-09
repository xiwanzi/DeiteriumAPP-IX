package objectstorage

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func verifyChatFixture(data []byte, thumbnail bool) (ChatImage, error) {
	h := sha256.Sum256(data)
	m := md5.Sum(data)
	return VerifyChatImage(data, hex.EncodeToString(h[:]), base64.StdEncoding.EncodeToString(m[:]), thumbnail)
}

// Optional cross-runtime samples exported by the actual Android and browser
// encoders; regular unit tests use the checked-in small fixtures above.
func TestChatMediaEncodedClientSamples(t *testing.T) {
	roots := filepath.SplitList(os.Getenv("DEUTERIUM_CHAT_MEDIA_FIXTURES"))
	if len(roots) == 0 {
		t.Skip("client sample directories not supplied")
	}
	count := 0
	for _, root := range roots {
		files, err := filepath.Glob(filepath.Join(root, "*.webp"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range files {
			name := filepath.Base(path)
			if !strings.Contains(name, ".image.") && !strings.Contains(name, ".thumb.") && !strings.Contains(name, ".thumbnail.") {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			thumbnail := strings.Contains(name, ".thumb.") || strings.Contains(name, ".thumbnail.")
			if _, err = verifyChatFixture(data, thumbnail); err != nil {
				t.Fatalf("client output rejected: %s: %v", name, err)
			}
			count++
		}
	}
	if count < 4 {
		t.Fatal("missing client samples", count)
	}
	t.Logf("validated %d client-encoded images/previews", count)
}
func TestChatWebPStaticAnimatedAndCorruption(t *testing.T) {
	for _, name := range []string{"sticker.webp", "motion.webp"} {
		data, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		v, err := verifyChatFixture(data, false)
		if err != nil {
			t.Fatal(name, err)
		}
		if name == "motion.webp" && (!v.Animated || v.Frames != 12 || v.DurationMS != 960) {
			t.Fatalf("animation metadata %#v", v)
		}
		if name == "sticker.webp" && (v.Animated || v.Width != 512 || v.Height != 512) {
			t.Fatal("static metadata", v)
		}
		if _, err = verifyChatFixture(data[:len(data)-1], false); err == nil {
			t.Fatal("truncation accepted")
		}
		if _, err = VerifyChatImage(data, "wrong", "wrong", false); err == nil {
			t.Fatal("wrong digest accepted")
		}
		if _, err = verifyChatFixture(data, true); err == nil {
			t.Fatal("oversized/animated thumbnail accepted")
		}
	}
}
func TestChatWebPRejectsAnimatedPixelAndDurationBombs(t *testing.T) {
	source, err := os.ReadFile("testdata/motion.webp")
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func([]byte){
		func(b []byte) {
			for p := 12; p+8 < len(b); {
				n := int(binary.LittleEndian.Uint32(b[p+4 : p+8]))
				if string(b[p:p+4]) == "VP8X" {
					b[p+12] = 255
					b[p+13] = 255
					return
				}
				p += 8 + n + n%2
			}
		},
		func(b []byte) {
			for p := 12; p+8 < len(b); {
				n := int(binary.LittleEndian.Uint32(b[p+4 : p+8]))
				if string(b[p:p+4]) == "ANMF" {
					b[p+20] = 255
					b[p+21] = 255
					b[p+22] = 1
					return
				}
				p += 8 + n + n%2
			}
		},
		func(b []byte) { b[20] &^= 2 },
	} {
		data := append([]byte(nil), source...)
		mutate(data)
		if _, err = verifyChatFixture(data, false); err == nil {
			t.Fatal("malformed animation accepted")
		}
	}
}
