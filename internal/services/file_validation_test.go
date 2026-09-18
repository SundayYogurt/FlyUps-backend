package services

import (
	"bytes"
	"mime/multipart"
	"testing"
)

func TestValidateUploadBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, filename string
		content        []byte
		invalid        bool
	}{
		{"empty", "photo.png", nil, true},
		{"wrong extension", "photo.jpg", pngMagic, true},
		{"forged image", "photo.png", []byte("plain text"), true},
		{"invalid workbook", "data.xlsx", []byte("PK\x03\x04not a zip"), true},
		{"image boundary", "photo.png", append(append([]byte{}, pngMagic...), make([]byte, 5*1024*1024-len(pngMagic))...), false},
		{"image oversized", "photo.png", append(append([]byte{}, pngMagic...), make([]byte, 5*1024*1024+1-len(pngMagic))...), true},
		{"video", "video.webm", []byte("\x1a\x45\xdf\xa3webm"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := newTestFile(tc.content)
			err := ValidateUpload(file, &multipart.FileHeader{Filename: tc.filename, Size: 1})
			if (err != nil) != tc.invalid {
				t.Fatalf("error %v, want invalid %v", err, tc.invalid)
			}
			if !tc.invalid {
				buffer := make([]byte, len(tc.content))
				_, _ = file.Read(buffer)
				if !bytes.Equal(buffer, tc.content) {
					t.Fatal("file cursor was not reset")
				}
			}
		})
	}
}
