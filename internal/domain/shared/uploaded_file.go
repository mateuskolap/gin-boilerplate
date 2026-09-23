package shared

import "io"

type UploadedFile struct {
	Content io.Reader
	Size    int64
}
