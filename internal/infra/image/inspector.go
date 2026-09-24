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
	"gin-boilerplate/internal/domain/shared"
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
	_, format, err := stdimage.DecodeConfig(inspectedSource)
	if err != nil {
		return port.InspectedImage{}, fmt.Errorf("inspect image: %w: %v", port.ErrInvalidImage, err)
	}
	if err := ctx.Err(); err != nil {
		return port.InspectedImage{}, err
	}

	imageFormat, ok := shared.ImageFormatFromDecoder(format)
	if !ok {
		return port.InspectedImage{}, port.ErrUnsupportedImageFormat
	}

	return port.InspectedImage{
		Content:   io.MultiReader(bytes.NewReader(consumed.Bytes()), src),
		Extension: imageFormat.Extension,
	}, nil
}
