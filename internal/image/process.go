package image

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrium/goheif"
	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
)

type ProcessedImage struct {
	Data     []byte
	Filename string
	MimeType string
}

// ProcessImage processes an image file: converts HEIC to JPG, strips EXIF data
func ProcessImage(imagePath string) (*ProcessedImage, error) {
	// Read the original file
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(imagePath))
	baseName := strings.TrimSuffix(filepath.Base(imagePath), ext)

	var img image.Image
	var outputExt string
	var mimeType string

	// Handle HEIC files
	if ext == ".heic" || ext == ".heif" {
		img, err = goheif.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to decode HEIC image: %w", err)
		}
		outputExt = ".jpg"
		mimeType = "image/jpeg"
		baseName = baseName + "_converted"
	} else {
		// For other image types, decode and re-encode to strip EXIF
		img, _, err = image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}

		// Determine output format based on input
		switch ext {
		case ".jpg", ".jpeg":
			outputExt = ".jpg"
			mimeType = "image/jpeg"
		case ".png":
			outputExt = ".png"
			mimeType = "image/png"
		default:
			// Default to JPEG for unknown formats
			outputExt = ".jpg"
			mimeType = "image/jpeg"
		}
	}

	// Re-encode the image (this strips EXIF data)
	var buf bytes.Buffer
	if outputExt == ".png" {
		err = imaging.Encode(&buf, img, imaging.PNG)
	} else {
		err = imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(95))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return &ProcessedImage{
		Data:     buf.Bytes(),
		Filename: baseName + outputExt,
		MimeType: mimeType,
	}, nil
}

// HasEXIF checks if an image file has EXIF data
func HasEXIF(imagePath string) (bool, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = exif.Decode(f)
	if err != nil {
		if err == io.EOF || strings.Contains(err.Error(), "exif") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
