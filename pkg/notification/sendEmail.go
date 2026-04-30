package notification

import (
	"flyup/config"
	"fmt"

	"github.com/resend/resend-go/v2"
)

type NotificationClient interface {
	SendVerifyEmail(to string, verifyLink string) error
	SendResetPasswordEmail(to string, resetLink string) error
	SendMeetingEmail(to string, title string, date string, time string, meetingType string, link *string, place *string) error
	SendUpdateMeetingEmail(to string, title string, date string, time string, meetingType string, link *string, place *string, status string) error
	SendMilestoneVoteResultEmail(to, projectTitle string, phaseNo int, phaseTitle string, approved bool) error
	SendUserSuspendedEmail(to string, reason string) error
}

type notificationClient struct {
	config config.AppConfig
	client *resend.Client
}

func (n notificationClient) SendVerifyEmail(to string, verifyLink string) error {
	params := &resend.SendEmailRequest{
		From:    n.config.EmailFrom,
		To:      []string{to},
		Subject: "Verify your email",
		Html: `
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
</head>
<body style="margin:0;padding:0;background:#f6f9fc;font-family:Arial,Helvetica,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f6f9fc;padding:40px 0;">
    <tr>
      <td align="center">
        <table width="500" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:10px;padding:40px;text-align:center;">
          
          <!-- Logo -->
          <tr>
            <td style="padding-bottom:20px;">
              <img src="https://drive.google.com/uc?export=view&id=1yVzLRSBcGAG0c2eLfaSFNOd9MmLFUKUP">
            </td>
          </tr>

          <!-- Title -->
          <tr>
            <td style="font-size:24px;font-weight:bold;color:#333;padding-bottom:10px;">
              Verify your email
            </td>
          </tr>

          <!-- Text -->
          <tr>
            <td style="font-size:16px;color:#555;padding-bottom:30px;">
              Thanks for signing up for <strong>FlyUp</strong>.<br>
              Please confirm your email address to activate your account.
            </td>
          </tr>

          <!-- Button -->
          <tr>
            <td>
              <a href="` + verifyLink + `" 
                 style="background:#2563eb;color:#ffffff;text-decoration:none;padding:14px 28px;border-radius:6px;font-size:16px;font-weight:bold;display:inline-block;">
                 Verify Email
              </a>
            </td>
          </tr>

          <!-- Footer text -->
          <tr>
            <td style="font-size:13px;color:#888;padding-top:30px;">
              If you didn’t create this account, you can safely ignore this email.
            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td style="font-size:12px;color:#aaa;padding-top:20px;">
              © 2026 FlyUp. All rights reserved.
            </td>
          </tr>

        </table>
      </td>
    </tr>
  </table>
</body>
</html>
`,
	}

	_, err := n.client.Emails.Send(params)

	return err
}

func (n notificationClient) SendResetPasswordEmail(to string, resetLink string) error {

	params := &resend.SendEmailRequest{
		From:    n.config.EmailFrom,
		To:      []string{to},
		Subject: "Reset your password",
		Html: `
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
</head>
<body style="margin:0;padding:0;background:#f6f9fc;font-family:Arial,Helvetica,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f6f9fc;padding:40px 0;">
    <tr>
      <td align="center">
        <table width="500" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:10px;padding:40px;text-align:center;">
          
          <!-- Logo -->
          <tr>
            <td style="padding-bottom:20px;">
              <img src="https://drive.google.com/uc?export=view&id=1yVzLRSBcGAG0c2eLfaSFNOd9MmLFUKUP">
            </td>
          </tr>

          <!-- Title -->
          <tr>
            <td style="font-size:24px;font-weight:bold;color:#333;padding-bottom:10px;">
              Reset your password
            </td>
          </tr>

          <!-- Text -->
          <tr>
            <td style="font-size:16px;color:#555;padding-bottom:30px;">
              We received a request to reset your <strong>FlyUp</strong> password.<br>
              Click the button below to set a new password.
            </td>
          </tr>

          <!-- Button -->
          <tr>
            <td>
              <a href="` + resetLink + `" 
                 style="background:#2563eb;color:#ffffff;text-decoration:none;padding:14px 28px;border-radius:6px;font-size:16px;font-weight:bold;display:inline-block;">
                 Reset Password
              </a>
            </td>
          </tr>

          <!-- Expire -->
          <tr>
            <td style="font-size:13px;color:#888;padding-top:20px;">
              This link will expire in 30 minutes.
            </td>
          </tr>

          <!-- Ignore -->
          <tr>
            <td style="font-size:13px;color:#888;padding-top:10px;">
              If you didn’t request a password reset, you can safely ignore this email.
            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td style="font-size:12px;color:#aaa;padding-top:20px;">
              © 2026 FlyUp. All rights reserved.
            </td>
          </tr>

        </table>
      </td>
    </tr>
  </table>
</body>
</html>
`,
	}

	_, err := n.client.Emails.Send(params)

	return err
}

func (n notificationClient) SendMeetingEmail(to string, title string, date string, time string, meetingType string, link *string, place *string) error {

	var detail string

	if meetingType == "online" && link != nil {
		detail = `
		<tr>
			<td style="font-size:16px;color:#555;padding-bottom:20px;">
				Meeting Type: Online<br>
				Link: <a href="` + *link + `">Join Meeting</a>
			</td>
		</tr>`
	} else if meetingType == "onsite" && place != nil {
		detail = `
		<tr>
			<td style="font-size:16px;color:#555;padding-bottom:20px;">
				Meeting Type: Onsite<br>
				Location: ` + *place + `
			</td>
		</tr>`
	}

	params := &resend.SendEmailRequest{
		From:    n.config.EmailFrom,
		To:      []string{to},
		Subject: "Meeting Invitation",
		Html: `
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
</head>
<body style="margin:0;padding:0;background:#f6f9fc;font-family:Arial,Helvetica,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f6f9fc;padding:40px 0;">
    <tr>
      <td align="center">
        <table width="500" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:10px;padding:40px;text-align:center;">
          
          <!-- Logo -->
          <tr>
            <td style="padding-bottom:20px;">
              <img src="https://drive.google.com/uc?export=view&id=1yVzLRSBcGAG0c2eLfaSFNOd9MmLFUKUP">
            </td>
          </tr>

          <!-- Title -->
          <tr>
            <td style="font-size:24px;font-weight:bold;color:#333;padding-bottom:10px;">
              ` + title + `
            </td>
          </tr>

          <!-- Date Time -->
          <tr>
            <td style="font-size:16px;color:#555;padding-bottom:20px;">
              Date: ` + date + `<br>
              Time: ` + time + `
            </td>
          </tr>


			<!-- Voting Notice -->
			<tr>
  			<td style="font-size:14px;color:#92400e;background:#fef3c7;padding:12px;border-radius:6px;">
    			<b>Note:</b> This meeting includes a voting session. Please be prepared to participate.
 			 </td>
			</tr>

          ` + detail + `

          <!-- Button -->
          <tr>
            <td>
              <a href="` + func() string {
			if link != nil {
				return *link
			}
			return "#"
		}() + `" 
                 style="background:#2563eb;color:#ffffff;text-decoration:none;padding:14px 28px;border-radius:6px;font-size:16px;font-weight:bold;display:inline-block;">
                 View Details
              </a>
            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td style="font-size:12px;color:#aaa;padding-top:20px;">
              © 2026 FlyUp. All rights reserved.
            </td>
          </tr>

        </table>
      </td>
    </tr>
  </table>
</body>
</html>
`,
	}

	_, err := n.client.Emails.Send(params)
	return err
}

func (n notificationClient) SendUpdateMeetingEmail(to string, title string, date string, time string, meetingType string, link *string, place *string, status string, // "updated" | "canceled"
) error {

	var detail string

	// detail
	if meetingType == "online" && link != nil {
		detail = `
		<tr>
			<td style="font-size:16px;color:#555;padding-bottom:20px;">
				Meeting Type: Online<br>
				Link: <a href="` + *link + `">Join Meeting</a>
			</td>
		</tr>`
	} else if meetingType == "onsite" && place != nil {
		detail = `
		<tr>
			<td style="font-size:16px;color:#555;padding-bottom:20px;">
				Meeting Type: Onsite<br>
				Location: ` + *place + `
			</td>
		</tr>`
	} else if meetingType == "hybrid" {
		detail = `
		<tr>
			<td style="font-size:16px;color:#555;padding-bottom:20px;">
				Meeting Type: Hybrid<br>
				` + func() string {
			if link != nil {
				return "Link: <a href=\"" + *link + "\">Join Meeting</a><br>"
			}
			return ""
		}() + `
				` + func() string {
			if place != nil {
				return "Location: " + *place
			}
			return ""
		}() + `
			</td>
		</tr>`
	}

	// title + badge
	var header string
	if status == "canceled" {
		header = "Meeting Canceled"
	} else {
		header = "Meeting Updated"
	}

	params := &resend.SendEmailRequest{
		From:    n.config.EmailFrom,
		To:      []string{to},
		Subject: header,
		Html: `
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
</head>
<body style="margin:0;padding:0;background:#f6f9fc;font-family:Arial,Helvetica,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f6f9fc;padding:40px 0;">
    <tr>
      <td align="center">
        <table width="500" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:10px;padding:40px;text-align:center;">
          
          <!-- Title -->
          <tr>
            <td style="font-size:24px;font-weight:bold;color:#333;padding-bottom:10px;">
              ` + header + `
            </td>
          </tr>

          <!-- Description -->
          <tr>
            <td style="font-size:16px;color:#555;padding-bottom:20px;">
              ` + func() string {
			if status == "canceled" {
				return "This meeting has been canceled."
			}
			return "The meeting details have been updated."
		}() + `
            </td>
          </tr>

          <!-- Date Time -->
          <tr>
            <td style="font-size:16px;color:#555;padding-bottom:20px;">
              Date: ` + date + `<br>
              Time: ` + time + `
            </td>
          </tr>

          ` + detail + `

          <!-- Footer -->
          <tr>
            <td style="font-size:12px;color:#aaa;padding-top:20px;">
              © 2026 FlyUp. All rights reserved.
            </td>
          </tr>

        </table>
      </td>
    </tr>
  </table>
</body>
</html>
`,
	}

	_, err := n.client.Emails.Send(params)
	return err
}

func (n notificationClient) SendMilestoneVoteResultEmail(to, projectTitle string, phaseNo int, phaseTitle string, approved bool) error {
	resultText := "ผ่านการโหวต ✅"
	resultColor := "#16a34a"
	descText := "Milestone นี้ได้รับการยืนยันจากนักลงทุน และกำลังดำเนินการปล่อยทุนให้แก่ Pioneer"
	if !approved {
		resultText = "ไม่ผ่านการโหวต ❌"
		resultColor = "#dc2626"
		descText = "Milestone นี้ไม่ผ่านการโหวต ทีม Pioneer จะต้องปรับปรุงและส่งใหม่"
	}

	params := &resend.SendEmailRequest{
		From:    n.config.EmailFrom,
		To:      []string{to},
		Subject: fmt.Sprintf("ผลการโหวต Milestone Phase %d — %s", phaseNo, projectTitle),
		Html: fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background:#f6f9fc;font-family:Arial,Helvetica,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f6f9fc;padding:40px 0;">
    <tr><td align="center">
      <table width="500" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:10px;padding:40px;text-align:center;">
        <tr><td style="padding-bottom:20px;">
          <img src="https://drive.google.com/uc?export=view&id=1yVzLRSBcGAG0c2eLfaSFNOd9MmLFUKUP">
        </td></tr>
        <tr><td style="font-size:22px;font-weight:bold;color:#333;padding-bottom:8px;">ผลการโหวต Milestone</td></tr>
        <tr><td style="font-size:15px;color:#555;padding-bottom:20px;">
          <b>%s</b> — Phase %d: %s
        </td></tr>
        <tr><td style="font-size:20px;font-weight:bold;color:%s;padding:16px 24px;border-radius:8px;background:#f9fafb;display:inline-block;">
          %s
        </td></tr>
        <tr><td style="font-size:14px;color:#666;padding-top:16px;padding-bottom:8px;">%s</td></tr>
        <tr><td style="font-size:12px;color:#aaa;padding-top:24px;">© 2026 FlyUp. All rights reserved.</td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`, projectTitle, phaseNo, phaseTitle, resultColor, resultText, descText),
	}

	_, err := n.client.Emails.Send(params)
	return err
}

func (n notificationClient) SendUserSuspendedEmail(to string, reason string) error {

	params := &resend.SendEmailRequest{
		From:    n.config.EmailFrom,
		To:      []string{to},
		Subject: "Your account has been suspended",
		Html: `
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
</head>
<body style="margin:0;padding:0;background:#f6f9fc;font-family:Arial,Helvetica,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f6f9fc;padding:40px 0;">
    <tr>
      <td align="center">
        <table width="500" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:10px;padding:40px;text-align:center;">
          
          <!-- Title -->
          <tr>
            <td style="font-size:24px;font-weight:bold;color:#dc2626;padding-bottom:10px;">
              Account Suspended
            </td>
          </tr>

          <!-- Message -->
          <tr>
            <td style="font-size:16px;color:#555;padding-bottom:20px;">
              Your <strong>FlyUp</strong> account has been suspended due to the following reason:
            </td>
          </tr>

          <!-- Reason -->
          <tr>
            <td style="font-size:15px;color:#111;background:#f3f4f6;padding:15px;border-radius:6px;">
              ` + reason + `
            </td>
          </tr>

          <!-- Info -->
          <tr>
            <td style="font-size:14px;color:#666;padding-top:20px;">
              If you believe this is a mistake, please contact our support team.
            </td>
          </tr>

          <!-- Footer -->
          <tr>
            <td style="font-size:12px;color:#aaa;padding-top:30px;">
              © 2026 FlyUp. All rights reserved.
            </td>
          </tr>

        </table>
      </td>
    </tr>
  </table>
</body>
</html>
`,
	}

	_, err := n.client.Emails.Send(params)
	return err
}

func NewNotificationClient(config config.AppConfig) NotificationClient {
	client := resend.NewClient(config.ResendAPIKey)

	return &notificationClient{
		config: config,
		client: client,
	}
}
