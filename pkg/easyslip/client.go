package easyslip

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Recipient struct{ Bank, Account string }
type Verification struct {
	TransRef      string
	SenderBank    string
	TransferredAt time.Time
}
type Verifier interface {
	Verify(imageURL string, amount float64, recipient Recipient) (*Verification, error)
}
type Client struct {
	APIKey, CloudName string
	HTTP              *http.Client
	endpoint          string
}

func New(apiKey, cloudName string) *Client {
	return &Client{APIKey: apiKey, CloudName: cloudName, endpoint: "https://api.easyslip.com/v2/verify/bank", HTTP: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

// Account matching must use a full account number, never masked digits or names alone.
func accountNumber(value string) string {
	return strings.NewReplacer("-", "", " ", "", "\t", "").Replace(value)
}
func validAccount(value string) bool {
	if len(value) < 6 || len(value) > 20 {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
func bankMatches(expected, id, short, name string) bool {
	expected = strings.ToUpper(strings.TrimSpace(expected))
	if expected == "" {
		return false
	}
	for _, candidate := range []string{id, short, name} {
		candidate = strings.ToUpper(strings.TrimSpace(candidate))
		if candidate != "" && (expected == candidate || strings.HasSuffix(expected, "("+candidate+")")) {
			return true
		}
	}
	// Profile selections may include a short-code suffix, while the provider
	// returns only the Thai name (or omits the word "bank").
	nameOnly := func(value string) string {
		value = strings.ToUpper(strings.TrimSpace(strings.SplitN(value, "(", 2)[0]))
		return strings.TrimSpace(strings.TrimPrefix(value, "ธนาคาร"))
	}
	if nameOnly(name) != "" && nameOnly(expected) == nameOnly(name) {
		return true
	}
	return false
}
func (c *Client) Verify(imageURL string, amount float64, recipient Recipient) (*Verification, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, errors.New("ยังไม่ได้ตั้งค่า EASYSLIP_API_KEY")
	}
	account := accountNumber(recipient.Account)
	if !validAccount(account) || strings.TrimSpace(recipient.Bank) == "" {
		return nil, errors.New("ยังไม่ได้ตั้งค่าบัญชีผู้รับสำหรับตรวจสลิป")
	}
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount <= 0 || math.Abs(amount*100-math.Round(amount*100)) > 0.0001 {
		return nil, errors.New("ยอดโอนต้องมากกว่า 0 และมีทศนิยมไม่เกิน 2 ตำแหน่ง")
	}
	u, err := url.Parse(imageURL)
	if err != nil || len(imageURL) > 2048 || u.Scheme != "https" || u.Host != "res.cloudinary.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || c.CloudName == "" || !strings.HasPrefix(u.Path, "/"+c.CloudName+"/image/upload/") {
		return nil, errors.New("กรุณาแนบรูปสลิปที่อัปโหลดผ่านระบบ")
	}
	body, _ := json.Marshal(map[string]any{"url": imageURL, "matchAmount": amount, "matchAccount": true, "checkDuplicate": false})
	// Duplicate consumption is committed locally with the business transaction. A
	// provider duplicate flag alone would prevent retry after a database failure.
	req, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("ไม่สามารถเตรียมตรวจสลิปได้")
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, errors.New("ติดต่อ EasySlip ไม่สำเร็จ กรุณาลองใหม่")
	}
	defer resp.Body.Close()
	var result struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
		Data struct {
			IsDuplicate     bool  `json:"isDuplicate"`
			IsAmountMatched *bool `json:"isAmountMatched"`
			MatchedAccount  *struct {
				BankNumber string                                   `json:"bankNumber"`
				Bank       struct{ Code, ShortCode, NameTh string } `json:"bank"`
			} `json:"matchedAccount"`
			RawSlip struct {
				TransRef string `json:"transRef"`
				Date     string `json:"date"`
				Amount   struct {
					Amount float64 `json:"amount"`
				} `json:"amount"`
				Sender struct {
					Bank struct {
						ID string `json:"id"`
					} `json:"bank"`
				} `json:"sender"`
				Receiver struct {
					Bank    struct{ ID, Short, Name string } `json:"bank"`
					Account struct {
						Bank *struct{ Type, Account string } `json:"bank"`
					} `json:"account"`
				} `json:"receiver"`
			} `json:"rawSlip"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return nil, errors.New("ผลตอบกลับ EasySlip ไม่ถูกต้อง กรุณาลองใหม่")
	}
	if resp.StatusCode != http.StatusOK || !result.Success {
		switch result.Error.Code {
		case "SLIP_PENDING":
			return nil, errors.New("สลิปยังรอข้อมูลจากธนาคาร กรุณารอสักครู่แล้วลองใหม่")
		case "SLIP_NOT_FOUND":
			return nil, errors.New("ไม่พบรายการโอนจากสลิป กรุณาตรวจรูปและ QR Code")
		case "QUOTA_EXCEEDED", "INSUFFICIENT_BALANCE":
			return nil, errors.New("โควต้า EasySlip ไม่เพียงพอ กรุณาติดต่อผู้ดูแลระบบ")
		default:
			return nil, errors.New("EasySlip ตรวจสลิปไม่สำเร็จ กรุณาตรวจการตั้งค่าหรือลองใหม่")
		}
	}
	d := result.Data
	if d.IsDuplicate {
		return nil, errors.New("สลิปนี้ถูกใช้แล้ว")
	}
	if d.IsAmountMatched != nil && !*d.IsAmountMatched || math.Abs(d.RawSlip.Amount.Amount-amount) > 0.0001 {
		return nil, errors.New("ยอดเงินในสลิปไม่ตรงกับยอดที่ต้องจ่าย")
	}
	matched := false
	if m := d.MatchedAccount; m != nil {
		matched = accountNumber(m.BankNumber) == account && bankMatches(recipient.Bank, m.Bank.Code, m.Bank.ShortCode, m.Bank.NameTh)
	} else if b := d.RawSlip.Receiver.Account.Bank; b != nil && b.Type == "BANKAC" {
		matched = accountNumber(b.Account) == account && bankMatches(recipient.Bank, d.RawSlip.Receiver.Bank.ID, d.RawSlip.Receiver.Bank.Short, d.RawSlip.Receiver.Bank.Name)
	}
	if !matched {
		return nil, errors.New("บัญชีผู้รับในสลิปไม่ตรง หรือยืนยันเลขบัญชีเต็มไม่ได้ กรุณาลงทะเบียนบัญชีผู้รับกับ EasySlip")
	}
	date, err := time.Parse(time.RFC3339, d.RawSlip.Date)
	if err != nil || d.RawSlip.TransRef == "" || d.RawSlip.Sender.Bank.ID == "" || date.After(time.Now().Add(5*time.Minute)) {
		return nil, errors.New("ข้อมูลอ้างอิงหรือเวลาของสลิปไม่ถูกต้อง")
	}
	return &Verification{TransRef: d.RawSlip.TransRef, SenderBank: d.RawSlip.Sender.Bank.ID, TransferredAt: date}, nil
}
