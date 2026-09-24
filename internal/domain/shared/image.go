package shared

import "io"

type ImageStream struct {
	Content     io.ReadCloser
	ContentType string
	Size        int64
}
