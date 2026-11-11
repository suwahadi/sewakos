package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/davidbyttow/govips/v2/vips"
)

func testProcessor(maxPixels int64) *Processor {
	return NewProcessor(Config{MaxPixels: maxPixels, LargePx: 1600, ThumbPx: 400, LargeQuality: 80, ThumbQuality: 75, Concurrency: 2})
}

func solid(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	return img
}

func encodePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, solid(width, height)); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// exifSegment builds a little-endian APP1 Exif block with an Orientation tag and a GPS IFD holding GPSLatitudeRef.
func exifSegment(orientation uint16) []byte {
	le := binary.LittleEndian
	tiff := []byte("II*\x00")
	tiff = le.AppendUint32(tiff, 8)
	tiff = le.AppendUint16(tiff, 2)
	tiff = le.AppendUint16(tiff, 0x0112)
	tiff = le.AppendUint16(tiff, 3)
	tiff = le.AppendUint32(tiff, 1)
	tiff = le.AppendUint16(tiff, orientation)
	tiff = le.AppendUint16(tiff, 0)
	gpsOffset := uint32(8 + 2 + 2*12 + 4)
	tiff = le.AppendUint16(tiff, 0x8825)
	tiff = le.AppendUint16(tiff, 4)
	tiff = le.AppendUint32(tiff, 1)
	tiff = le.AppendUint32(tiff, gpsOffset)
	tiff = le.AppendUint32(tiff, 0)
	tiff = le.AppendUint16(tiff, 1)
	tiff = le.AppendUint16(tiff, 0x0001)
	tiff = le.AppendUint16(tiff, 2)
	tiff = le.AppendUint32(tiff, 2)
	tiff = append(tiff, 'N', 0, 0, 0)
	tiff = le.AppendUint32(tiff, 0)

	payload := append([]byte("Exif\x00\x00"), tiff...)
	segment := []byte{0xFF, 0xE1}
	segment = binary.BigEndian.AppendUint16(segment, uint16(len(payload)+2))
	return append(segment, payload...)
}

func encodeJPEGWithExif(t *testing.T, width, height int, orientation uint16) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, solid(width, height), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	plain := buf.Bytes()
	return append(append(append([]byte{}, plain[:2]...), exifSegment(orientation)...), plain[2:]...)
}

func webpChunks(t *testing.T, data []byte) []string {
	t.Helper()
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		t.Fatalf("output is not a WebP file")
	}
	var chunks []string
	for offset := 12; offset+8 <= len(data); {
		size := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		chunks = append(chunks, string(data[offset:offset+4]))
		offset += 8 + size + size%2
	}
	return chunks
}

func dimensions(t *testing.T, data []byte) (int, int) {
	t.Helper()
	img, err := vips.NewImageFromBuffer(data)
	if err != nil {
		t.Fatal(err)
	}
	defer img.Close()
	return img.Width(), img.Height()
}

func TestProcessRotatesByExifAndStripsMetadata(t *testing.T) {
	out, err := testProcessor(40_000_000).Process(t.Context(), encodeJPEGWithExif(t, 200, 100, 6))
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if out.Width != 100 || out.Height != 200 {
		t.Errorf("large = %dx%d, want 100x200 after the EXIF rotation", out.Width, out.Height)
	}
	for _, encoded := range [][]byte{out.Large, out.Thumb} {
		for _, chunk := range webpChunks(t, encoded) {
			if chunk == "EXIF" || chunk == "XMP " {
				t.Errorf("WebP still carries a %q chunk", chunk)
			}
		}
	}
}

func TestProcessFitsWithoutUpscaling(t *testing.T) {
	processor := testProcessor(40_000_000)
	cases := []struct {
		name                           string
		input                          []byte
		largeW, largeH, thumbW, thumbH int
	}{
		{"large png", encodePNG(t, 2000, 1000), 1600, 800, 400, 200},
		{"small png", encodePNG(t, 300, 150), 300, 150, 300, 150},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := processor.Process(t.Context(), tc.input)
			if err != nil {
				t.Fatalf("Process: %v", err)
			}
			if w, h := dimensions(t, out.Large); w != tc.largeW || h != tc.largeH || out.Width != w || out.Height != h {
				t.Errorf("large = %dx%d (reported %dx%d), want %dx%d", w, h, out.Width, out.Height, tc.largeW, tc.largeH)
			}
			if w, h := dimensions(t, out.Thumb); w != tc.thumbW || h != tc.thumbH {
				t.Errorf("thumb = %dx%d, want %dx%d", w, h, tc.thumbW, tc.thumbH)
			}
		})
	}
}

func TestProcessAcceptsWebpInput(t *testing.T) {
	source, err := vips.NewImageFromBuffer(encodePNG(t, 800, 600))
	if err != nil {
		t.Fatal(err)
	}
	input, _, err := source.ExportWebp(&vips.WebpExportParams{Quality: 90})
	source.Close()
	if err != nil {
		t.Fatal(err)
	}

	out, err := testProcessor(40_000_000).Process(t.Context(), input)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if w, h := dimensions(t, out.Thumb); w != 400 || h != 300 {
		t.Errorf("thumb = %dx%d, want 400x300", w, h)
	}
}

func TestProcessRejectsUnusableInput(t *testing.T) {
	cases := map[string]struct {
		input     []byte
		maxPixels int64
	}{
		"corrupt jpeg":    {append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x42}, 64)...), 40_000_000},
		"not an image":    {[]byte("hello, this is plain text"), 40_000_000},
		"too many pixels": {encodePNG(t, 30, 30), 800},
		"png header only": {[]byte("\x89PNG\r\n\x1a\n"), 40_000_000},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := testProcessor(tc.maxPixels).Process(t.Context(), tc.input)
			if !errors.Is(err, ErrUnprocessable) {
				t.Errorf("err = %v, want ErrUnprocessable", err)
			}
		})
	}
}
