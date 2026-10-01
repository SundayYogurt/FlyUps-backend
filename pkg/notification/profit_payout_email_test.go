package notification

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProfitPayoutEmailContent(t *testing.T) {
	subject, body := profitPayoutEmail("Siam <Spirit>", 2, 20000, "REF<&123", true)
	require.NotContains(t, subject, "[ทดสอบ]")
	require.Contains(t, subject, "ไตรมาสที่ 2")
	require.Contains(t, body, "Siam &lt;Spirit&gt;")
	require.Contains(t, body, "฿20000.00")
	require.Contains(t, body, "REF&lt;&amp;123")
	require.Contains(t, body, "ไม่ได้ยืนยันว่ามีเงินเข้าบัญชีจริง")
	subject, body = profitPayoutEmail("Siam Spirit", 2, 20000, "REF-123", false)
	require.NotContains(t, subject, "[ทดสอบ]")
	require.NotContains(t, body, "รายการทดสอบ")
}
