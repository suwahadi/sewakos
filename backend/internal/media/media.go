package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/davidbyttow/govips/v2/vips"
)

var ErrUnprocessable = errors.New("image cannot be processed")

type Config struct {
	MaxPixels    int64
	LargePx      int
	ThumbPx      int
	LargeQuality int
	ThumbQuality int
	Concurrency  int
}

type Image struct {
	Large  []byte
	Thumb  []byte
	Width  int
	Height int
}

type Processor struct {
	cfg   Config
	slots chan struct{}
}

var startVips sync.Once

func NewProcessor(cfg Config) *Processor {
	startVips.Do(func() {
		vips.LoggingSettings(nil, vips.LogLevelError)
		_ = vips.Startup(nil)
	})
	return &Processor{cfg: cfg, slots: make(chan struct{}, max(cfg.Concurrency, 1))}
}

func IsSupportedImage(data []byte) bool {
	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return true
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return true
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return true
	}
	return false
}

func (p *Processor) Process(ctx context.Context, data []byte) (Image, error) {
	if !IsSupportedImage(data) {
		return Image{}, ErrUnprocessable
	}
	select {
	case p.slots <- struct{}{}:
	case <-ctx.Done():
		return Image{}, fmt.Errorf("wait for image slot: %w", ctx.Err())
	}
	defer func() { <-p.slots }()

	// libvips reads only the header here; pixels are decoded later by the thumbnail calls.
	header, err := vips.NewImageFromBuffer(data)
	if err != nil {
		return Image{}, fmt.Errorf("%w: %v", ErrUnprocessable, err)
	}
	pixels := int64(header.Width()) * int64(header.Height())
	header.Close()
	if pixels > p.cfg.MaxPixels {
		return Image{}, fmt.Errorf("%w: %d pixels", ErrUnprocessable, pixels)
	}

	large, width, height, err := encodeWebp(data, p.cfg.LargePx, p.cfg.LargeQuality)
	if err != nil {
		return Image{}, err
	}
	thumb, _, _, err := encodeWebp(data, p.cfg.ThumbPx, p.cfg.ThumbQuality)
	if err != nil {
		return Image{}, err
	}
	return Image{Large: large, Thumb: thumb, Width: width, Height: height}, nil
}

func encodeWebp(data []byte, box, quality int) ([]byte, int, int, error) {
	img, err := vips.NewThumbnailWithSizeFromBuffer(data, box, box, vips.InterestingNone, vips.SizeDown)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("%w: %v", ErrUnprocessable, err)
	}
	defer img.Close()
	out, _, err := img.ExportWebp(&vips.WebpExportParams{Quality: quality, StripMetadata: true})
	if err != nil {
		return nil, 0, 0, fmt.Errorf("%w: %v", ErrUnprocessable, err)
	}
	return stripWebpMetadata(out), img.Width(), img.Height(), nil
}

// libvips 8.14.0-8.14.1 (Debian bookworm) ignores webpsave's "strip" and copies the source EXIF,
// so EXIF/XMP chunks are cut from the RIFF container and their VP8X feature flags cleared.
func stripWebpMetadata(data []byte) []byte {
	const exifFlag, xmpFlag = 0x08, 0x04
	out := append([]byte{}, data[:12]...)
	for offset := 12; offset+8 <= len(data); {
		size := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		end := min(offset+8+size+size%2, len(data))
		switch chunk := data[offset:end]; string(chunk[:4]) {
		case "EXIF", "XMP ":
		case "VP8X":
			flagged := append([]byte{}, chunk...)
			flagged[8] &^= exifFlag | xmpFlag
			out = append(out, flagged...)
		default:
			out = append(out, chunk...)
		}
		offset = end
	}
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
	return out
}
