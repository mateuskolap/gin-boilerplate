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
	Extension string
}

// ImageInspector identifies supported image formats from their contents.
type ImageInspector interface {
	// Inspect identifies the format of src and returns a reader that yields the
	// complete original content, including bytes consumed during inspection.
	Inspect(ctx context.Context, src io.Reader) (InspectedImage, error)
}
