package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	TargetVolume            float64 `yaml:"target_volume,omitempty"`
	CastIp                  string  `yaml:"cast_ip,omitempty"`
	FileToCast              string  `yaml:"file_to_cast,omitempty"`
	SftpRemote              string  `yaml:"sftp_remote,omitempty"`
	CameraManufacturer      string  `yaml:"camera_manufacturer,omitempty"`
	RcloneConfigFile        string  `yaml:"rclone_config_file,omitempty"`
	CastDeviceName          string  `yaml:"cast_device_name,omitempty"`
	WorkerCount             int     `yaml:"worker_count,omitempty"`
	CastAudioOnCompletion   bool    `yaml:"cast_audio_on_completion,omitempty"`
	DeleteImagesAfterUpload bool    `yaml:"delete_images_after_upload,omitempty"`
}

func ParseConfig(configFile string) (*Config, error) {
	yamlData, err := os.OpenFile(configFile, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, fmt.Errorf("unable to read config file at :%s - %s", configFile, err)
	}

	var config Config
	decoder := yaml.NewDecoder(yamlData)

	// Enable strict mode: trigger error on unknown fields
	decoder.KnownFields(true)

	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("Validation Error: %v\n", err)
	}
	return &config, err
}
