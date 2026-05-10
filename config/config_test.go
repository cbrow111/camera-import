package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

const testConfigFile = `test_config.yaml`

const (
	expectedTargetVolume            = 0.80
	expectedCastIp                  = "192.168.1.0"
	expectedFileToCast              = "/file/to/cast"
	expectedSftpRemote              = "sftpgo"
	expectedRcloneConfigFile        = "/config/rclone.conf"
	expectedCastDeviceName          = "Kitchen Speaker"
	expectedCastAudioOnCompletion   = true
	expectedDeleteImagesAfterUpload = true
	expectedWorkerCount             = 4
	expectedCameraManufacturer      = "Canon"
)

func TestConfig(t *testing.T) {
	yamlData, err := os.OpenFile(testConfigFile, os.O_RDONLY|os.O_CREATE, 0666)
	assert.NoError(t, err, "data should be read without an error")

	var config Config
	decoder := yaml.NewDecoder(yamlData)

	// Enable strict mode: trigger error on unknown fields
	decoder.KnownFields(true)

	err = decoder.Decode(&config)
	if err != nil {
		fmt.Printf("Validation Error: %v\n", err)
	} else {
		fmt.Printf("Parsed Config: %+v\n", config)
	}
	assert.Equal(t, config.TargetVolume, expectedTargetVolume, "target volume should be equal")
	assert.Equal(t, config.CastIp, expectedCastIp, "cast ip should be equal")
	assert.Equal(t, config.FileToCast, expectedFileToCast, "file to cast should be equal")
	assert.Equal(t, config.SftpRemote, expectedSftpRemote, "sftp remote should be equal")
	assert.Equal(t, config.CameraManufacturer, expectedCameraManufacturer, "camera model should be equal")
	assert.Equal(t, config.RcloneConfigFile, expectedRcloneConfigFile, "rclone config file should be equal")
	assert.Equal(t, config.CastDeviceName, expectedCastDeviceName, "camera model should be equal")
	assert.Equal(t, config.WorkerCount, expectedWorkerCount, "worker count should be equal")
	assert.Equal(t, config.CastAudioOnCompletion, expectedCastAudioOnCompletion, "cast audio on-completion should be equal")
	assert.Equal(t, config.DeleteImagesAfterUpload, expectedDeleteImagesAfterUpload, "delete images after upload should be equal")
}
