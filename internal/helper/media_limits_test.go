package helper

import (
	"flyup/internal/domain"
	"flyup/internal/dto"
	"strings"
	"testing"
)

func TestMediaLimits(t *testing.T) {
	var items []domain.ProjectMedia
	var urls []string
	for i := 0; i < 5; i++ {
		items = append(items, domain.ProjectMedia{URL: "https://example.com/image.png", Type: []domain.MediaType{domain.MediaTypeImage}}, domain.ProjectMedia{URL: "https://example.com/video.mp4", Type: []domain.MediaType{domain.MediaTypeVideo}})
		urls = append(urls, "https://example.com/image.png", "https://example.com/video.mp4")
	}
	if err := ValidateProjectMedia(items); err != nil {
		t.Fatal(err)
	}
	if err := ValidateInputLimits(dto.UpdateMilestoneRequest{URLs: urls}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateInputLimits(dto.SubmitMilestoneRequest{Attachments: urls}); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []domain.MediaType{domain.MediaTypeImage, domain.MediaTypeVideo} {
		var six []domain.ProjectMedia
		var sixURLs []string
		ext := ".png"
		if kind == domain.MediaTypeVideo {
			ext = ".mp4"
		}
		for i := 0; i < 6; i++ {
			six = append(six, domain.ProjectMedia{URL: "https://example.com/a" + ext, Type: []domain.MediaType{kind}})
			sixURLs = append(sixURLs, "https://example.com/a"+ext)
		}
		if ValidateProjectMedia(six) == nil {
			t.Fatalf("accepted six %s files", kind)
		}
		if ValidateMediaURLs(sixURLs) == nil {
			t.Fatalf("accepted six %s URLs", kind)
		}
	}
	for _, item := range []domain.ProjectMedia{
		{URL: "https://example.com/a.png", Type: []domain.MediaType{domain.MediaTypeVideo}},
		{URL: "https://example.com/a.png", Type: []domain.MediaType{domain.MediaTypeImage, domain.MediaTypeVideo}},
		{URL: "https://example.com/a.png", Type: []domain.MediaType{"invalid"}},
		{URL: "javascript:alert(1)", Type: []domain.MediaType{domain.MediaTypeImage}},
	} {
		if ValidateProjectMedia([]domain.ProjectMedia{item}) == nil {
			t.Fatalf("accepted invalid media: %+v", item)
		}
	}
}

func TestAdditionalInputLimits(t *testing.T) {
	for _, value := range []any{
		dto.UpdateProjectRequest{Title: stringPointer(" \t")},
		dto.UpdateProjectRequest{Title: stringPointer("a\x00b")},
		struct {
			ID uint64 `json:"id"`
		}{^uint64(0)},
		struct {
			Body string `json:"body"`
		}{strings.Repeat("a", 10001)},
	} {
		if ValidateInputLimits(value) == nil {
			t.Errorf("accepted invalid input: %T", value)
		}
	}
}
