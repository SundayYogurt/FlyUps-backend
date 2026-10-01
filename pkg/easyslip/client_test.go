package easyslip

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVerify(t *testing.T) {
	for _, tc := range []struct{ name, change, want string }{
		{"matched recipient", "", ""},
		{"wrong amount", "amount", "ยอดเงิน"},
		{"wrong account", "account", "บัญชีผู้รับ"},
		{"wrong bank", "bank", "บัญชีผู้รับ"},
		{"masked without match", "masked", "บัญชีผู้รับ"},
		{"full unregistered account", "full", ""},
		{"token is not account", "token", "บัญชีผู้รับ"},
		{"duplicate", "duplicate", "ถูกใช้แล้ว"},
		{"missing ref", "ref", "ข้อมูลอ้างอิง"},
		{"missing bank", "sender", "ข้อมูลอ้างอิง"},
		{"invalid date", "date", "ข้อมูลอ้างอิง"},
		{"pending", "pending", "รอข้อมูล"},
		{"missing key", "key", "API_KEY"},
		{"external image", "url", "อัปโหลดผ่านระบบ"},
		{"fractional satang", "fraction", "ทศนิยม"},
		{"malformed response", "json", "ผลตอบกลับ"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r.Header.Get("Authorization") != "Bearer secret" {
					t.Error("missing auth")
				}
				var request map[string]any
				_ = json.NewDecoder(r.Body).Decode(&request)
				if request["matchAccount"] != true || request["matchAmount"] != float64(100) || request["checkDuplicate"] != false {
					t.Error("wrong verification options")
				}
				if tc.change == "json" {
					_, _ = w.Write([]byte("invalid"))
					return
				}
				if tc.change == "pending" {
					w.WriteHeader(404)
					_, _ = w.Write([]byte(`{"success":false,"error":{"code":"SLIP_PENDING"}}`))
					return
				}
				matched := map[string]any{"bankNumber": "063-8-72573-8", "bank": map[string]any{"code": "004", "shortCode": "KBANK"}}
				account := map[string]any{"bank": map[string]any{"type": "BANKAC", "account": "xxx-x-72573-8"}}
				raw := map[string]any{"transRef": "VERIFIED-123", "date": time.Now().Add(-time.Hour).Format(time.RFC3339), "sender": map[string]any{"bank": map[string]any{"id": "014"}}, "amount": map[string]any{"amount": 100}, "receiver": map[string]any{"bank": map[string]any{"id": "004", "short": "KBANK"}, "account": account}}
				data := map[string]any{"isAmountMatched": true, "isDuplicate": false, "matchedAccount": matched, "rawSlip": raw}
				switch tc.change {
				case "amount":
					raw["amount"] = map[string]any{"amount": 101}
				case "account":
					matched["bankNumber"] = "0638725739"
				case "bank":
					matched["bank"] = map[string]any{"code": "014", "shortCode": "SCB"}
				case "masked":
					data["matchedAccount"] = nil
				case "full":
					data["matchedAccount"] = nil
					account["bank"] = map[string]any{"type": "BANKAC", "account": "0638725738"}
				case "token":
					data["matchedAccount"] = nil
					account["bank"] = map[string]any{"type": "TOKEN", "account": "0638725738"}
				case "duplicate":
					data["isDuplicate"] = true
				case "ref":
					raw["transRef"] = ""
				case "sender":
					raw["sender"] = nil
				case "date":
					raw["date"] = "bad"
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
			}))
			defer server.Close()
			client := New("secret", "flyup")
			client.endpoint = server.URL
			image := "https://res.cloudinary.com/flyup/image/upload/slip.png"
			amount := float64(100)
			if tc.change == "key" {
				client.APIKey = ""
			}
			if tc.change == "url" {
				image = "https://attacker.test/slip.png"
			}
			if tc.change == "fraction" {
				amount = 100.001
			}
			got, err := client.Verify(image, amount, Recipient{Bank: "ธนาคารกสิกรไทย (KBANK)", Account: "0638725738"})
			if tc.want == "" {
				if err != nil || got.TransRef != "VERIFIED-123" {
					t.Fatalf("got %v, %v", got, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want %s, got %v", tc.want, err)
			}
			if (tc.change == "key" || tc.change == "url" || tc.change == "fraction") && called {
				t.Fatal("invalid input called provider")
			}
		})
	}
}
