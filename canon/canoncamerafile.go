package canon

import "github.com/jonmol/gphoto2"

type CanonCameraFile struct {
	*gphoto2.CameraFilePath
}

func (c CanonCameraFile) Name() string {
	return c.CameraFilePath.Name
}

func (c CanonCameraFile) Folder() string {
	return c.CameraFilePath.Folder
}
