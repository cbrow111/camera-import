package main

import (
	"camera-import/camera"
	_ "camera-import/canon"
	"camera-import/config"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/rclone/rclone/fs/config/configfile"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

var rootCmd = cobra.Command{
	Use:   "import-images [config file]",
	Short: "Import images from camera and upload to ftp",
	Long: "import-images is a cli tool to import images from a camera, upload them to an ftp server, and optionally " +
		"play a sound on a chromecast capable device on completion",

	Args: cobra.ArbitraryArgs,
	Run:  execute,
}

var configPath string

func init() {
	confDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	defaultPath := filepath.Join(confDir, "import-images", "config.yaml")
	rootCmd.Flags().StringVarP(&configPath, "configPath", "c", defaultPath, "path to config file")
}

func execute(cmd *cobra.Command, args []string) {
	configfile.Install()

	appConfig, err := config.ParseConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	factory, ok := camera.CameraRegistry[strings.ToLower(appConfig.CameraManufacturer)]
	if !ok {
		log.Printf("unknown camera model: %s", appConfig.CameraManufacturer)
		log.Printf("valid camera models are: %s", strings.Join(maps.Keys(camera.CameraRegistry), ", "))
		return
	}
	cmra, err := factory(appConfig)
	if err != nil {
		log.Printf("error creating %s camera: %v", appConfig.CameraManufacturer, err)
		return
	}
	log.Printf("creating %s camera: %#v", appConfig.CameraManufacturer, cmra)

	ctx := context.Background()
	err = camera.StartUploader(ctx, cmra, appConfig)
	if err != nil {
		log.Printf("error occurred while uploading photos: %v", err)
		return
	}

	if appConfig.CastAudioOnCompletion {
		PlayCompletionSound(appConfig)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
