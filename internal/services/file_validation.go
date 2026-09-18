package services

import (
	"archive/zip"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// ValidateUpload checks the actual file before any external upload occurs.
func ValidateUpload(file multipart.File, header *multipart.FileHeader) error {
	if file == nil || header == nil || strings.TrimSpace(header.Filename) == "" || !utf8.ValidString(header.Filename) || strings.ContainsRune(header.Filename, 0) || utf8.RuneCountInString(header.Filename) > 255 {
		return fmt.Errorf("invalid filename (max 255 characters)")
	}
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	defer file.Seek(0, io.SeekStart)
	if size <= 0 {
		return fmt.Errorf("file must not be empty")
	}
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return err
	}
	mime := http.DetectContentType(buf[:n])
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]string{".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif", ".webp": "image/webp", ".pdf": "application/pdf", ".mp4": "video/mp4", ".webm": "video/webm", ".mov": "video/quicktime", ".avi": "video/avi"}
	max := int64(5 * 1024 * 1024)
	if strings.HasPrefix(mime, "video/") {
		max = 50 * 1024 * 1024
	}
	if size > max {
		return fmt.Errorf("file exceeds maximum size of %d MB", max/(1024*1024))
	}
	if ext == ".xlsx" && mime == "application/zip" {
		archive, err := zip.NewReader(file, size)
		if err != nil {
			return fmt.Errorf("invalid Excel file")
		}
		var contentTypes, workbook bool
		for _, f := range archive.File {
			contentTypes = contentTypes || f.Name == "[Content_Types].xml"
			workbook = workbook || f.Name == "xl/workbook.xml"
		}
		if !contentTypes || !workbook {
			return fmt.Errorf("invalid Excel workbook")
		}
		return nil
	}
	if ext == ".xls" && mime == "application/vnd.ms-excel" {
		return nil
	}
	if expected, ok := allowed[ext]; !ok || expected != mime {
		return fmt.Errorf("unsupported file type or extension does not match content")
	}
	return nil
}
