package helper

import "testing"

func TestNormalizeThaiIDNumber(t *testing.T) {
	for _, input := range []string{"1234567890121", "1-2345-67890-12-1", "1 2345 67890 12 1"} {
		got, err := NormalizeThaiIDNumber(input)
		if err != nil || got != "1234567890121" {
			t.Fatalf("normalize %q: got %q, error %v", input, got, err)
		}
	}
	for _, input := range []string{"1234567890123", "123456789012", "123456789012x", ""} {
		if got, err := NormalizeThaiIDNumber(input); err == nil {
			t.Fatalf("accepted invalid ID %q as %q", input, got)
		}
	}
}

func TestIDCardFingerprint(t *testing.T) {
	a := IDCardFingerprint("1234567890121", "secret-a")
	if len(a) != 64 || a != IDCardFingerprint("1234567890121", "secret-a") {
		t.Fatal("fingerprint is not stable")
	}
	if a == IDCardFingerprint("1234567890121", "secret-b") || a == IDCardFingerprint("1234567890122", "secret-a") {
		t.Fatal("fingerprint did not distinguish the secret or card number")
	}
}
