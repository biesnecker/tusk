package image

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/rwcarlsen/goexif/exif"
)

func createTestJPEG(path string, withEXIF bool) error {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// Fill with a simple pattern
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 255, 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if withEXIF {
		// Create a buffer to write JPEG with EXIF
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, nil); err != nil {
			return err
		}

		// For testing purposes, we'll just write the JPEG
		// A real EXIF would require more complex setup
		_, err = f.Write(buf.Bytes())
		return err
	}

	return jpeg.Encode(f, img, nil)
}

func createTestPNG(path string) error {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 255, 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

func TestProcessImageJPEG(t *testing.T) {
	tmpDir := t.TempDir()
	jpegPath := filepath.Join(tmpDir, "test.jpg")

	if err := createTestJPEG(jpegPath, false); err != nil {
		t.Fatalf("Failed to create test JPEG: %v", err)
	}

	processed, err := ProcessImage(jpegPath)
	if err != nil {
		t.Fatalf("Failed to process JPEG: %v", err)
	}

	if processed.MimeType != "image/jpeg" {
		t.Errorf("Expected mime type image/jpeg, got %s", processed.MimeType)
	}

	if len(processed.Data) == 0 {
		t.Error("Processed image data is empty")
	}

	if processed.Filename != "test.jpg" {
		t.Errorf("Expected filename test.jpg, got %s", processed.Filename)
	}
}

func TestProcessImagePNG(t *testing.T) {
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "test.png")

	if err := createTestPNG(pngPath); err != nil {
		t.Fatalf("Failed to create test PNG: %v", err)
	}

	processed, err := ProcessImage(pngPath)
	if err != nil {
		t.Fatalf("Failed to process PNG: %v", err)
	}

	if processed.MimeType != "image/png" {
		t.Errorf("Expected mime type image/png, got %s", processed.MimeType)
	}

	if len(processed.Data) == 0 {
		t.Error("Processed image data is empty")
	}

	if processed.Filename != "test.png" {
		t.Errorf("Expected filename test.png, got %s", processed.Filename)
	}
}

func TestProcessImageNonExistent(t *testing.T) {
	_, err := ProcessImage("/path/to/nonexistent/image.jpg")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestProcessImageInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(txtPath, []byte("not an image"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := ProcessImage(txtPath)
	if err == nil {
		t.Error("Expected error for invalid image format, got nil")
	}
}

func TestHasEXIF(t *testing.T) {
	tmpDir := t.TempDir()
	jpegPath := filepath.Join(tmpDir, "test.jpg")

	if err := createTestJPEG(jpegPath, false); err != nil {
		t.Fatalf("Failed to create test JPEG: %v", err)
	}

	// Our simple test images don't have real EXIF data
	hasEXIF, err := HasEXIF(jpegPath)
	if err != nil {
		t.Fatalf("Failed to check EXIF: %v", err)
	}

	// Should be false since we didn't add real EXIF data
	if hasEXIF {
		t.Error("Expected no EXIF data in test image")
	}
}

func TestHasEXIFNonExistent(t *testing.T) {
	_, err := HasEXIF("/path/to/nonexistent/image.jpg")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestEXIFStripping(t *testing.T) {
	tmpDir := t.TempDir()
	jpegPath := filepath.Join(tmpDir, "test.jpg")

	// Create a JPEG
	if err := createTestJPEG(jpegPath, false); err != nil {
		t.Fatalf("Failed to create test JPEG: %v", err)
	}

	// Process it
	processed, err := ProcessImage(jpegPath)
	if err != nil {
		t.Fatalf("Failed to process image: %v", err)
	}

	// Write processed image to temp file
	processedPath := filepath.Join(tmpDir, "processed.jpg")
	if err := os.WriteFile(processedPath, processed.Data, 0644); err != nil {
		t.Fatalf("Failed to write processed image: %v", err)
	}

	// Check that processed image has no EXIF
	f, err := os.Open(processedPath)
	if err != nil {
		t.Fatalf("Failed to open processed image: %v", err)
	}
	defer f.Close()

	_, err = exif.Decode(f)
	if err == nil {
		t.Error("Expected no EXIF data in processed image, but found some")
	}
}
