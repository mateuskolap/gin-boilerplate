package image

import (
	"bytes"
	"context"
	"errors"
	stdimage "image"
	"image/color"
	"image/gif"
	"image/png"
	"io"
	"testing"

	"gin-boilerplate/internal/domain/port"
)

func encodedPNG(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, stdimage.NewRGBA(stdimage.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestInspectorRecognizesSupportedFormatsAndPreservesBytes(t *testing.T) {
	source := encodedPNG(t)
	inspected, err := NewInspector().Inspect(context.Background(), bytes.NewReader(source))
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}
	content, err := io.ReadAll(inspected.Content)
	if err != nil || inspected.Extension != ".png" || !bytes.Equal(content, source) {
		t.Fatalf("Inspect() extension=%q content matches=%v error=%v", inspected.Extension, bytes.Equal(content, source), err)
	}
}

func TestInspectorRejectsInvalidUnsupportedAndCanceledInput(t *testing.T) {
	inspector := NewInspector()
	if _, err := inspector.Inspect(context.Background(), nil); !errors.Is(err, port.ErrInvalidImage) {
		t.Fatalf("Inspect(nil) error = %v", err)
	}
	if _, err := inspector.Inspect(context.Background(), bytes.NewReader([]byte("not an image"))); !errors.Is(err, port.ErrInvalidImage) {
		t.Fatalf("Inspect(invalid) error = %v", err)
	}

	var unsupported bytes.Buffer
	gifImage := stdimage.NewPaletted(stdimage.Rect(0, 0, 1, 1), color.Palette{color.Black, color.White})
	if err := gif.Encode(&unsupported, gifImage, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := inspector.Inspect(context.Background(), bytes.NewReader(unsupported.Bytes())); !errors.Is(err, port.ErrUnsupportedImageFormat) {
		t.Fatalf("Inspect(GIF) error = %v, want unsupported format", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := inspector.Inspect(ctx, bytes.NewReader(encodedPNG(t))); !errors.Is(err, context.Canceled) {
		t.Fatalf("Inspect(canceled) error = %v", err)
	}
}
