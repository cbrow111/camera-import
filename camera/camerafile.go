package camera

import "io"

type DownloadableCameraFile interface {
	DownloadImage(buffer io.Writer, leaveOnCamera bool) error
	Folder() string
	Name() string
}
