package canon

import (
	"camera-import/camera"
	"camera-import/config"
	"fmt"
	"log"
	"strings"

	"github.com/jonmol/gphoto2"
)

func init() {
	camera.Register("canon", func(config *config.Config) (camera.DownloadableCamera, error) {
		return NewCanonCamera()
	})
}

type Camera struct {
	camera *gphoto2.Camera
}

func NewCanonCamera() (*Camera, error) {
	camera, err := gphoto2.NewCamera()
	if err != nil {
		if strings.Contains(err.Error(), "-105") {
			return nil, fmt.Errorf("no camera detected")
		}
		return nil, fmt.Errorf("could not create camera: %w", err)
	}

	modelSettingName := "cameramodel"
	modelNameSetting, err := camera.GetSetting(modelSettingName)
	if err != nil {
		return nil, fmt.Errorf("could not get setting: %s, %w", modelSettingName, err)
	}

	modelName, err := modelNameSetting.Get()
	if err != nil {
		return nil, fmt.Errorf("could not get model setting %s: %w", modelNameSetting, err)
	}

	log.Printf("successfully created camera: %s", modelName)
	return &Camera{camera: camera}, nil

}

func (r *Camera) ListPhotos() (camera.DownloadableCameraFiles, error) {
	// Canon R6 usually stores photos in /store_xxxx/DCIM/100CANON
	files, err := r.camera.ListFiles()
	if err != nil {
		return nil, err
	}

	allFiles := camera.CollectFiles(files, `cr3`)
	images := make(camera.DownloadableCameraFiles, 0, len(allFiles))
	for _, f := range allFiles {
		images = append(images, CanonCameraFile{f})
	}

	return images, nil
}

func (r *Camera) DeletePhoto(img camera.DownloadableCameraFile) error {
	if canonImage, ok := img.(CanonCameraFile); ok {
		return r.camera.DeleteFile(canonImage.CameraFilePath)
	}
	return fmt.Errorf("cannot delete image %s from unexpectedType %T", img.Name(), img)
}

func (r *Camera) RegistryName() string {
	return "Canon_EOS"
}
