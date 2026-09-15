package helper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"
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

// helper function เอาไว้โหลดรูปทีละใบแล้วยัดใส่ writer
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

	// บังคับ filename .jpg + content-type image/jpeg
	// (Cloudinary ส่ง byte เป็น jpg มาแล้วจาก f_jpg แต่ iApp ดูนามสกุล/ctype)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s.jpg"`, fieldName, fieldName))
	h.Set("Content-Type", "image/jpeg")

	log.Printf("[iApp] %s → ctype=%s len=%s url=%s",
		fieldName,
		resp.Header.Get("Content-Type"),
		resp.Header.Get("Content-Length"),
		imageURL)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", fieldName, err)
	}

	// RIFF....WEBP
	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return fmt.Errorf("%s is still webp, cloudinary transform failed: %s", fieldName, imageURL)
	}
	// JPEG ขึ้นต้นด้วย FF D8 FF
	if len(data) < 3 || data[0] != 0xFF || data[1] != 0xD8 {
		return fmt.Errorf("%s is not a valid jpeg: %s", fieldName, imageURL)
	}

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

	// สลับ field! iApp คาดหวังให้ file0 เป็นรูป Selfie และ file1 เป็นรูปบัตรประชาชน
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
