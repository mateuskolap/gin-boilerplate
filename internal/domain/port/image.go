package port

import (
	"context"
	"errors"
	"io"
)

var (
	ErrInvalidImage           = errors.New("invalid image")
	ErrUnsupportedImageFormat = errors.New("unsupported image format")
)

type InspectedImage struct {
	Content   io.Reader
	Format    string
	Extension string
	Width     int
	Height    int
}

// ImageInspector identifies an image from its contents and returns a reader
// containing every byte from the original input, including bytes read during
// inspection.
type ImageInspector interface {
	Inspect(ctx context.Context, src io.Reader) (InspectedImage, error)
}
