package shared

import (
	"path"
	"strings"
)

// ImageFormat describes the file extension and content type for a supported image.
type ImageFormat struct {
	Extension   string
	ContentType string
}

// ImageFormatFromDecoder maps an image.DecodeConfig format to its stored metadata.
func ImageFormatFromDecoder(format string) (ImageFormat, bool) {
	switch format {
	case "jpeg":
		return ImageFormat{Extension: ".jpg", ContentType: "image/jpeg"}, true
	case "png":
		return ImageFormat{Extension: ".png", ContentType: "image/png"}, true
	default:
		return ImageFormat{}, false
	}
}

// ImageFormatFromKey reads the format from a storage key's file extension.
func ImageFormatFromKey(key string) (ImageFormat, bool) {
	switch strings.ToLower(path.Ext(key)) {
	case ".jpg", ".jpeg":
		return ImageFormatFromDecoder("jpeg")
	case ".png":
		return ImageFormatFromDecoder("png")
	default:
		return ImageFormat{}, false
	}
}
