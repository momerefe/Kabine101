package benefitimage

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func sample(t *testing.T, format string) []byte {
	t.Helper()
	im := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	im.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 90, B: 40, A: 255})
	var out bytes.Buffer
	var err error
	if format == "png" {
		err = png.Encode(&out, im)
	} else {
		err = jpeg.Encode(&out, im, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestNormalizePNGAndJPEGStripsMetadata(t *testing.T) {
	for _, format := range []string{"png", "jpeg"} {
		t.Run(format, func(t *testing.T) {
			data := sample(t, format)
			metadata := []byte("private-marker-EXIF-<script>alert(1)</script>")
			if format == "png" {
				var chunk bytes.Buffer
				payload := append([]byte("Comment\x00"), metadata...)
				_ = binary.Write(&chunk, binary.BigEndian, uint32(len(payload)))
				chunk.WriteString("tEXt")
				chunk.Write(payload)
				_ = binary.Write(&chunk, binary.BigEndian, crc32.ChecksumIEEE(chunk.Bytes()[4:]))
				data = append(append(append([]byte{}, data[:len(data)-12]...), chunk.Bytes()...), data[len(data)-12:]...)
			} else {
				segment := []byte{0xff, 0xe1, 0, byte(len(metadata) + 4)}
				segment = append(segment, metadata...)
				segment = append(segment, 0xff, 0xd9) // Marker bytes inside a sized APP segment are metadata.
				data = append(append(append([]byte{}, data[:2]...), segment...), data[2:]...)
			}
			asset, issue := Normalize(base64.StdEncoding.EncodeToString(data))
			if issue != nil {
				t.Fatalf("valid %s rejected: %+v", format, issue)
			}
			if asset.ContentType != "image/"+format || len(asset.Data) == 0 || bytes.Contains(asset.Data, metadata) {
				t.Fatalf("image metadata retained or wrong MIME: %s", asset.ContentType)
			}
			decoded, actual, err := image.Decode(bytes.NewReader(asset.Data))
			if err != nil || actual != format || decoded.Bounds().Dx() != 3 || decoded.Bounds().Dy() != 2 {
				t.Fatalf("invalid normalized image: %v", err)
			}
		})
	}
}

func TestNormalizeRejectsInvalidAndPolyglotImages(t *testing.T) {
	pngData, jpegData := sample(t, "png"), sample(t, "jpeg")
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"SVG", []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)},
		{"truncated PNG", pngData[:len(pngData)-8]},
		{"truncated JPEG", jpegData[:len(jpegData)-2]},
		{"PNG with HTML suffix", append(append([]byte{}, pngData...), []byte("<script>payload</script>")...)},
		{"JPEG with fake EOI suffix", append(append([]byte{}, jpegData...), []byte("<script>payload</script>\xff\xd9")...)},
		{"PNG with prefix", append([]byte("<svg>"), pngData...)},
		{"GIF", []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			asset, issue := Normalize(base64.StdEncoding.EncodeToString(tc.data))
			if issue == nil || issue.Status != 400 || len(asset.Data) != 0 {
				t.Fatalf("invalid image accepted: %+v", issue)
			}
		})
	}
	for _, raw := range []string{"", "not base64!", "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngData)} {
		if _, issue := Normalize(raw); issue == nil || issue.Status != 400 {
			t.Fatalf("invalid base64 accepted: %q", raw)
		}
	}
}

func TestNormalizeRejectsOversizedInputAndDimensionsBeforeDecode(t *testing.T) {
	if _, issue := Normalize(base64.StdEncoding.EncodeToString(make([]byte, MaxBytes+1))); issue == nil || issue.Status != 413 {
		t.Fatalf("oversized decoded input accepted: %+v", issue)
	}
	for _, size := range [][2]uint32{{8193, 1}, {1, 8193}, {4001, 4000}} {
		data := sample(t, "png")
		binary.BigEndian.PutUint32(data[16:20], size[0])
		binary.BigEndian.PutUint32(data[20:24], size[1])
		binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
		if _, issue := Normalize(base64.StdEncoding.EncodeToString(data)); issue == nil || issue.Status != 400 || !strings.Contains(issue.Message, "16 milyon") {
			t.Fatalf("dimensions %v not rejected before decoding: %+v", size, issue)
		}
	}
	var out boundedBuffer
	if _, err := out.Write(make([]byte, MaxBytes)); err != nil {
		t.Fatal(err)
	}
	if _, err := out.Write([]byte{1}); err != errEncodedSize || out.Len() != MaxBytes {
		t.Fatal("normalized output exceeded its limit")
	}
}

func TestServePrivateRasterHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	data := sample(t, "png")
	Serve(w, "image/png", data)
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Content-Type") != "image/png" || w.Header().Get("Content-Length") != strconv.Itoa(len(data)) || !bytes.Equal(w.Body.Bytes(), data) {
		t.Fatal("private image response headers or bytes changed")
	}
}
