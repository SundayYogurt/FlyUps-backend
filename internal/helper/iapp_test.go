package helper

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
	"golang.org/x/image/webp"
)

func TestVerifyFaceAndIDCardSendsJPEGFromWebPWithProviderFieldOrder(t *testing.T) {
	// Fixture from golang.org/x/image/testdata (BSD-3-Clause; see testdata/LICENSE-go-image.txt).
	selfieWebP, err := os.ReadFile("testdata/gopher-doc.1bpp.lossless.webp")
	if err != nil {
		t.Fatal(err)
	}
	selfieImage, err := webp.Decode(bytes.NewReader(selfieWebP))
	if err != nil {
		t.Fatal(err)
	}
	idCardImage := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			idCardImage.Set(x, y, color.RGBA{R: 10, G: 80, B: 200, A: 255})
		}
	}
	var idCardPNG bytes.Buffer
	if err := png.Encode(&idCardPNG, idCardImage); err != nil {
		t.Fatal(err)
	}

	const cardURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/id.png"
	const selfieURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/selfie.webp"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder(http.MethodGet, cardURL, httpmock.NewBytesResponder(http.StatusOK, idCardPNG.Bytes()))
	httpmock.RegisterResponder(http.MethodGet, selfieURL, httpmock.NewBytesResponder(http.StatusOK, selfieWebP))
	posted := false
	httpmock.RegisterResponder(http.MethodPost, "https://api.iapp.co.th/v3/store/ekyc/face-and-id-card-verification", func(req *http.Request) (*http.Response, error) {
		posted = true
		if req.Header.Get("apikey") != "test-key" {
			t.Errorf("missing API key")
		}
		if err := req.ParseMultipartForm(11 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			return httpmock.NewStringResponse(400, "bad multipart"), nil
		}
		for field, want := range map[string]image.Rectangle{"file0": selfieImage.Bounds(), "file1": idCardImage.Bounds()} {
			files := req.MultipartForm.File[field]
			if len(files) != 1 {
				t.Errorf("%s has %d files", field, len(files))
				continue
			}
			header := files[0]
			if !strings.HasSuffix(header.Filename, ".jpg") || header.Header.Get("Content-Type") != "image/jpeg" {
				t.Errorf("%s metadata: %s %s", field, header.Filename, header.Header.Get("Content-Type"))
			}
			file, err := header.Open()
			if err != nil {
				t.Error(err)
				continue
			}
			payload, err := io.ReadAll(file)
			_ = file.Close()
			if err != nil {
				t.Error(err)
				continue
			}
			decoded, err := jpeg.Decode(bytes.NewReader(payload))
			if err != nil {
				t.Errorf("%s is not JPEG: %v", field, err)
				continue
			}
			if decoded.Bounds() != want {
				t.Errorf("%s dimensions %v, want %v", field, decoded.Bounds(), want)
			}
		}
		return httpmock.NewStringResponse(http.StatusOK, `{"total":{"isSamePerson":"false","confidence":40}}`), nil
	})
	result, err := NewIAppService("test-key").VerifyFaceAndIDCard(cardURL, selfieURL)
	if err != nil {
		t.Fatal(err)
	}
	if !posted || !strings.Contains(result, "isSamePerson") {
		t.Fatalf("iApp was not called correctly: posted=%v result=%q", posted, result)
	}
}
