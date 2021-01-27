package service

import (
	"bytes"
	"fmt"
	"os"
	"sync"

	"github.com/google/uuid"
)

// IMageStore is an interface to store laptop images
type ImageStore interface {
	Save(laptopID string, imageType string, imgData bytes.Buffer) (string, error)
}

type DiskImageStore struct {
	mutex       sync.Mutex
	imageFolder string
	images      map[string]*ImageInfo
}

type ImageInfo struct {
	LaptopID string
	Type     string
	Path     string
}

func NewDiskImageStore(imageFolder string) *DiskImageStore {
	return &DiskImageStore{
		imageFolder: imageFolder,
		images:      make(map[string]*ImageInfo),
	}
}

func (store *DiskImageStore) Save(
	laptopID string,
	imageType string,
	imageData bytes.Buffer,
) (string, error) {
	imageID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("cannot generate image id %w", err)
	}

	imagePath := fmt.Sprintf("%s/%s%s", store.imageFolder, imageID, imageType)
	file, err := os.Create(imagePath)
	if err != nil {
		return "", fmt.Errorf("erorr creating file %w", err)
	}

	_, err = imageData.WriteTo(file)
	if err != nil {
		return "", fmt.Errorf("erorr writingto file %w", err)
	}

	store.mutex.Lock()
	defer store.mutex.Unlock()

	store.images[imageID.String()] = &ImageInfo{
		LaptopID: laptopID,
		Type:     imageType,
		Path:     imagePath,
	}

	return imageID.String(), nil
}
