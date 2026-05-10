package camera

import "C"
import (
	"bytes"
	"camera-import/config"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/rclone/rclone/backend/all"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/operations"
)

// DownloadableCamera defines the behaviors needed from the Canon R6
type DownloadableCamera interface {
	ListPhotos() (DownloadableCameraFiles, error)
	DeletePhoto(img DownloadableCameraFile) error
	RegistryName() string
}

type DownloadableCameraFiles []DownloadableCameraFile

type FileBuffer struct {
	Image DownloadableCameraFile
	Data  []byte
}

func DownloadImages(jobs chan *FileBuffer, images DownloadableCameraFiles) error {
	var err error
	for _, img := range images {
		buff := bytes.Buffer{}
		err = img.DownloadImage(&buff, true)
		if err != nil {
			err = errors.Join(err, fmt.Errorf("could not download image: %s, %v", img, err))
			continue
		}
		jobs <- &FileBuffer{
			Image: img,
			Data:  buff.Bytes(),
		}
	}
	close(jobs)
	return err
}

// UploadBufferToRclone streams the memory buffer to your SFTPGo remote.
func UploadBufferToRclone(ctx context.Context, job *FileBuffer, toDelete chan DownloadableCameraFile, dstFs fs.Fs) error {
	// 2. Wrap the byte slice in a Reader
	reader := bytes.NewReader(job.Data)

	// 3. Wrap to satisfy io.ReadCloser
	// rclone's Rcat needs to call .Close() when the transfer finishes
	readCloser := io.NopCloser(reader)

	// 4. Stream to the remote using Rcat
	// Rcat(context, destinationFs, remotePath, reader, modificationTime, metadata)
	_, err := operations.Rcat(ctx, dstFs, job.Image.Name(), readCloser, time.Now(), nil)
	if err == nil {
		toDelete <- job.Image
	}
	return err
}

func StartUploader(ctx context.Context, camera DownloadableCamera, appConfig *config.Config) error {
	photos, err := camera.ListPhotos()
	if err != nil {
		return err
	}

	for _, photo := range photos {
		fmt.Println(filepath.Join(photo.Folder(), photo.Name()))
	}

	jobs := make(chan *FileBuffer, appConfig.WorkerCount)
	toDelete := make(chan DownloadableCameraFile, len(photos))
	var wg sync.WaitGroup

	// 1. Get the rclone destination
	dstFs, err := fs.NewFs(ctx, appConfig.SftpRemote+":")
	if err != nil {
		return err
	}

	// Start workers - Consumer
	for i := 0; i < appConfig.WorkerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				// Re-use the existing Rcat logic
				err := UploadBufferToRclone(ctx, job, toDelete, dstFs)
				if err != nil {
					fmt.Printf("Upload failed for %s: %v\n", job.Image.Name(), err)
				}
			}
		}()
	}

	// Producer
	err = DownloadImages(jobs, photos)
	if err != nil {
		return err
	}

	deleteDone := make(chan struct{})
	if appConfig.DeleteImagesAfterUpload {
		go func() {
			err = DeleteImages(ctx, camera, toDelete)
			close(deleteDone)
		}()
	} else {
		close(deleteDone)
	}

	wg.Wait()
	close(toDelete)
	<-deleteDone
	return err
}

func DeleteImages(ctx context.Context, camera DownloadableCamera, toDelete chan DownloadableCameraFile) error {
	var errs error
	for {
		select {
		case <-ctx.Done():
			return errors.Join(errs, ctx.Err())
		case img, ok := <-toDelete:
			if !ok {
				return errs
			}
			if err := camera.DeletePhoto(img); err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}
}
