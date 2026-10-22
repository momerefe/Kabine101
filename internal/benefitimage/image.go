// Package benefitimage validates uploaded raster images and serves normalized bytes.
package benefitimage

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"strconv"
)

const MaxBytes = 5 * 1024 * 1024
const MaxBodyBytes = 7*1024*1024 + 64*1024

type Asset struct {
	ContentType string
	Data        []byte
}
type Error struct {
	Status  int
	Message string
}

func invalid() *Error {
	return &Error{400, "Görsel okunamadı. Geçerli bir PNG veya JPEG dosyası seçin."}
}
func tooLarge() *Error {
	return &Error{413, "Görsel en fazla 5 MiB olabilir. Daha küçük bir görsel seçin."}
}

var errEncodedSize = errors.New("normalized image exceeds limit")

type boundedBuffer struct{ bytes.Buffer }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if len(p) > MaxBytes-b.Len() {
		return 0, errEncodedSize
	}
	return b.Buffer.Write(p)
}

// Normalize accepts raw base64, verifies dimensions before pixel allocation, then
// reencodes only decoded pixels. Original metadata and executable wrappers never persist.
func Normalize(raw string) (Asset, *Error) {
	if len(raw) > base64.StdEncoding.EncodedLen(MaxBytes) {
		return Asset{}, tooLarge()
	}
	data, err := base64.StdEncoding.Strict().DecodeString(raw)
	if err != nil || len(data) == 0 {
		return Asset{}, invalid()
	}
	if len(data) > MaxBytes {
		return Asset{}, tooLarge()
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") {
		return Asset{}, invalid()
	}
	if config.Width < 1 || config.Height < 1 || config.Width > 8192 || config.Height > 8192 || int64(config.Width)*int64(config.Height) > 16000000 {
		return Asset{}, &Error{400, "Görsel en fazla 8192 piksel genişliğinde ve yüksekliğinde, toplam 16 milyon piksel olabilir. Görseli küçültün."}
	}
	if format == "jpeg" && !completeJPEG(data) {
		return Asset{}, invalid()
	}
	r := bytes.NewReader(data)
	var decoded image.Image
	if format == "png" {
		decoded, err = png.Decode(r)
	} else {
		decoded, err = jpeg.Decode(r)
	}
	if err != nil || (format == "png" && r.Len() != 0) {
		return Asset{}, invalid()
	}
	var out boundedBuffer
	if format == "png" {
		err = png.Encode(&out, decoded)
	} else {
		err = jpeg.Encode(&out, decoded, &jpeg.Options{Quality: 85})
	}
	if errors.Is(err, errEncodedSize) {
		return Asset{}, tooLarge()
	}
	if err != nil {
		return Asset{}, invalid()
	}
	return Asset{ContentType: "image/" + format, Data: out.Bytes()}, nil
}

// completeJPEG walks marker lengths and scan byte stuffing to reject appended
// payloads, including fake EOI suffixes. Full image validity is checked by jpeg.Decode.
func completeJPEG(data []byte) bool {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return false
	}
	pos := 2
	for pos < len(data) {
		if data[pos] != 0xff {
			return false
		}
		for pos < len(data) && data[pos] == 0xff {
			pos++
		}
		if pos >= len(data) {
			return false
		}
		marker := data[pos]
		pos++
		if marker == 0xd9 {
			return pos == len(data)
		}
		if marker == 0x00 || marker == 0xd8 {
			return false
		}
		if marker == 0x01 || (marker >= 0xd0 && marker <= 0xd7) {
			continue
		}
		if pos+2 > len(data) {
			return false
		}
		size := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		if size < 2 || size > len(data)-pos {
			return false
		}
		pos += size
		if marker != 0xda {
			continue
		}
		for pos < len(data) {
			if data[pos] != 0xff {
				pos++
				continue
			}
			start := pos
			for pos < len(data) && data[pos] == 0xff {
				pos++
			}
			if pos >= len(data) {
				return false
			}
			if data[pos] == 0x00 || (data[pos] >= 0xd0 && data[pos] <= 0xd7) {
				pos++
				continue
			}
			pos = start
			break
		}
	}
	return false
}

func Serve(w http.ResponseWriter, contentType string, data []byte) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
