package helper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

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
	resp, err := http.Get(imageURL)
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

	part, err := writer.CreatePart(h)
	if err != nil {
		return fmt.Errorf("failed to create form part for %s: %v", fieldName, err)
	}

	if _, err = io.Copy(part, resp.Body); err != nil {
		return fmt.Errorf("failed to copy content for %s: %v", fieldName, err)
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
	client := &http.Client{}
	iAppResp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %v", err)
	}
	defer iAppResp.Body.Close()

	// อ่านและประมวลผล JSON Response จาก Iapp
	respBody, err := io.ReadAll(iAppResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// ถ้า status พัง
	if iAppResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("iapp api error (status: %d) : %s", iAppResp.StatusCode, string(respBody))
	}

	// เช็คว่าเป็น JSON String
	if !json.Valid(respBody) {
		return "", errors.New("invalid json response from iapp")
	}

	return string(respBody), nil
}
