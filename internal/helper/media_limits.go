package helper

import (
	"flyup/internal/domain"
	"fmt"
	"net/url"
	"path"
	"strings"
)

const MaxFilesPerType = 5

type InputError struct{ Message string }

func (e *InputError) Error() string { return e.Message }
func InvalidInput(format string, args ...any) error {
	return &InputError{Message: fmt.Sprintf(format, args...)}
}

// MediaURLType recognizes supported file extensions and Cloudinary resource paths.
// Unknown document URLs count as raw files.
func MediaURLType(raw string) domain.MediaType {
	u, err := url.Parse(raw)
	if err != nil {
		return domain.MediaTypeRaw
	}
	switch strings.ToLower(path.Ext(u.Path)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return domain.MediaTypeImage
	case ".mp4", ".webm", ".mov", ".avi":
		return domain.MediaTypeVideo
	}
	if u.Hostname() == "res.cloudinary.com" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 3 && (parts[1] == "image" || parts[1] == "video") {
			return domain.MediaType(parts[1])
		}
	}
	return domain.MediaTypeRaw
}

func ValidateMediaURLs(urls []string) error {
	if len(urls) > 10 {
		return InvalidInput("at most 10 files are allowed")
	}
	counts := map[domain.MediaType]int{}
	for _, raw := range urls {
		if strings.TrimSpace(raw) == "" {
			return InvalidInput("file URL must not be blank")
		}
		kind := MediaURLType(raw)
		counts[kind]++
		if counts[kind] > MaxFilesPerType {
			return InvalidInput("at most 5 %s files are allowed", kind)
		}
	}
	return nil
}

func ValidateProjectMedia(items []domain.ProjectMedia) error {
	if len(items) > 10 {
		return InvalidInput("at most 10 project media files are allowed")
	}
	counts := map[domain.MediaType]int{}
	for _, item := range items {
		if err := ValidateInputLimits(item); err != nil {
			return err
		}
		if strings.TrimSpace(item.URL) == "" {
			return InvalidInput("media URL is required")
		}
		if len(item.Type) != 1 {
			return InvalidInput("each media file must have exactly one type")
		}
		kind := item.Type[0]
		if kind != domain.MediaTypeImage && kind != domain.MediaTypeVideo && kind != domain.MediaTypeRaw {
			return InvalidInput("unsupported media type")
		}
		if inferred := MediaURLType(item.URL); inferred != domain.MediaTypeRaw && kind != inferred {
			return InvalidInput("media type does not match URL")
		}
		counts[kind]++
		if counts[kind] > MaxFilesPerType {
			return InvalidInput("at most 5 %s files are allowed", kind)
		}
	}
	return nil
}
