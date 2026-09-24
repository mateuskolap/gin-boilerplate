package image

import (
	"bytes"
	"context"
	"fmt"
	stdimage "image"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"gin-boilerplate/internal/domain/port"
)

const maxHeaderBytes int64 = 1 << 20

type Inspector struct{}

var _ port.ImageInspector = (*Inspector)(nil)

func NewInspector() *Inspector {
	return &Inspector{}
}

func (i *Inspector) Inspect(ctx context.Context, src io.Reader) (port.InspectedImage, error) {
	if src == nil {
		return port.InspectedImage{}, port.ErrInvalidImage
	}
	if err := ctx.Err(); err != nil {
		return port.InspectedImage{}, err
	}

	var consumed bytes.Buffer
	inspectedSource := io.TeeReader(io.LimitReader(src, maxHeaderBytes), &consumed)
	config, format, err := stdimage.DecodeConfig(inspectedSource)
	if err != nil {
		return port.InspectedImage{}, fmt.Errorf("inspect image: %w: %v", port.ErrInvalidImage, err)
	}
	if err := ctx.Err(); err != nil {
		return port.InspectedImage{}, err
	}

	extension, err := extensionFor(format)
	if err != nil {
		return port.InspectedImage{}, err
	}

	return port.InspectedImage{
		Content:   io.MultiReader(bytes.NewReader(consumed.Bytes()), src),
		Format:    format,
		Extension: extension,
		Width:     config.Width,
		Height:    config.Height,
	}, nil
}

func extensionFor(format string) (string, error) {
	switch format {
	case "jpeg":
		return ".jpg", nil
	case "png":
		return ".png", nil
	default:
		return "", port.ErrUnsupportedImageFormat
	}
}
