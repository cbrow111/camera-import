package camera

import (
	"fmt"
	"strings"

	"github.com/jonmol/gphoto2"
)

func CollectFiles(storage []gphoto2.CameraStorageInfo, fileType string) []*gphoto2.CameraFilePath {
	var files []*gphoto2.CameraFilePath

	var walk func(children []gphoto2.CameraFilePath)
	walk = func(children []gphoto2.CameraFilePath) {
		for i := range children {
			if children[i].Dir {
				walk(children[i].Children)
			} else {
				if strings.HasSuffix(strings.ToLower(children[i].Name), fmt.Sprintf(".%s", strings.ToLower(fileType))) {
					files = append(files, &children[i])
				}
			}
		}
	}

	for _, s := range storage {
		walk(s.Children)
	}
	return files
}
