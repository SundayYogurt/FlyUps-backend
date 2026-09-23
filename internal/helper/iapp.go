package helper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"

	_ "golang.org/x/image/webp"
)

type IAppError struct {
	StatusCode int
	Code       string // เช่น "INSUFFICIENT_CREDITS"
	Message    string
	Raw        string
}

type IAppService interface {
	VerifyFaceAndIDCard(idCardURL, selfieURL string) (string, error)
}

type iAppService struct {
	APIKey string
}

func NewIAppService(apiKey string) IAppService {
	return &iAppService{
		APIKey: apiKey,
	}
}

const maxIAppImageBytes = 10 * 1024 * 1024
const maxIAppImagePixels = 20_000_000

// Download the Cloudinary image and transcode it for iApp without changing the
// WebP asset used by the rest of the application.
func downloadAndAttachFile(writer *multipart.Writer, fieldName, imageURL string) error {
	var httpClient = &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Get(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", fieldName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s, status: %d", fieldName, resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxIAppImageBytes+1))
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", fieldName, err)
	}
	if len(data) > maxIAppImageBytes {
		return fmt.Errorf("%s exceeds iApp's 10 MB image limit", fieldName)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%s is not a supported image: %w", fieldName, err)
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxIAppImagePixels {
		return fmt.Errorf("%s image dimensions exceed the limit", fieldName)
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode %s: %w", fieldName, err)
	}
	if format != "jpeg" {
		bounds := decoded.Bounds()
		opaque := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		draw.Draw(opaque, opaque.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
		draw.Draw(opaque, opaque.Bounds(), decoded, bounds.Min, draw.Over)
		var converted bytes.Buffer
		if err := jpeg.Encode(&converted, opaque, &jpeg.Options{Quality: 85}); err != nil {
			return fmt.Errorf("encode %s as jpeg: %w", fieldName, err)
		}
		if converted.Len() > maxIAppImageBytes {
			return fmt.Errorf("converted %s exceeds iApp's 10 MB image limit", fieldName)
		}
		data = converted.Bytes()
	}
	log.Printf("[iApp] %s source=%s %dx%d outbound=jpeg (%d bytes)", fieldName, format, config.Width, config.Height, len(data))

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s.jpg"`, fieldName, fieldName))
	h.Set("Content-Type", "image/jpeg")

	part, err := writer.CreatePart(h)
	if err != nil {
		return fmt.Errorf("failed to create form part for %s: %w", fieldName, err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("failed to write content for %s: %w", fieldName, err)
	}

	return nil
}

func (s iAppService) VerifyFaceAndIDCard(idCardURL, selfieURL string) (string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// The live store endpoint expects the selfie holding the card in file0 and
	// the separate ID-card photo in file1 (confirmed with iApp's demo images).
	if err := downloadAndAttachFile(writer, "file0", selfieURL); err != nil {
		return "", err
	}

	if err := downloadAndAttachFile(writer, "file1", idCardURL); err != nil {
		return "", err
	}

	err := writer.Close()
	if err != nil {
		return "", err
	}

	// สร้าง HTTP Request ไป iApp Endpoint แบบ 2 รูป
	url := "https://api.iapp.co.th/v3/store/ekyc/face-and-id-card-verification"

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// แนบ Header APIKEY + Content-Type multipart
	req.Header.Set("apikey", s.APIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// ส่ง request ออกไป
	client := &http.Client{Timeout: 30 * time.Second}
	iAppResp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer iAppResp.Body.Close()

	// อ่านและประมวลผล JSON Response จาก Iapp
	respBody, err := io.ReadAll(iAppResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// ถ้า status พัง
	if iAppResp.StatusCode != http.StatusOK {
		var apiErr struct {
			Message   string `json:"message"`
			ErrorCode string `json:"error_code"`
		}
		_ = json.Unmarshal(respBody, &apiErr) // best-effort, ไม่ต้องสน error

		return "", &IAppError{
			StatusCode: iAppResp.StatusCode,
			Code:       apiErr.ErrorCode,
			Message:    apiErr.Message,
			Raw:        string(respBody),
		}
	}

	// เช็คว่าเป็น JSON String
	if !json.Valid(respBody) {
		return "", errors.New("invalid json response from iapp")
	}

	return string(respBody), nil
}

func (e *IAppError) Error() string {
	return fmt.Sprintf("iapp api error (status: %d): %s", e.StatusCode, e.Raw)
}

// true = ปัญหาฝั่งเรา/provider ไม่ใช่ความผิด user
func (e *IAppError) IsProviderUnavailable() bool {
	switch {
	case e.StatusCode == 402, // credits หมด
		e.StatusCode == 401, // key ผิด
		e.StatusCode == 429, // rate limit
		e.StatusCode >= 500: // provider ล่ม
		return true
	}
	return false
}

func IsProviderUnavailable(err error) bool {
	if err == nil {
		return false
	}

	var e *IAppError
	if errors.As(err, &e) {
		return e.IsProviderUnavailable()
	}

	// network error, timeout, DNS — นับเป็นฝั่งเราไม่พร้อมเหมือนกัน
	return true
}
