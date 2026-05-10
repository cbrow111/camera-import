package camera

import "camera-import/config"

var CameraRegistry = make(map[string]CameraFactory)

type CameraFactory func(config *config.Config) (DownloadableCamera, error)

// Register allows implementations to add themselves to the registry
func Register(name string, factory CameraFactory) {
	CameraRegistry[name] = factory
}
