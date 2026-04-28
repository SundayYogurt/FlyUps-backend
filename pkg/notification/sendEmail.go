package notification

import (
	"flyup/config"

	"github.com/resend/resend-go/v2"
)

type NotificationClient interface {
	SendVerifyEmail(to string, verifyLink string) error
	SendResetPasswordEmail(to string, resetLink string) error
	SendMeetingEmail(to string, title string, date string, time string, meetingType string, link *string, place *string) error
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

func NewNotificationClient(config config.AppConfig) NotificationClient {
	client := resend.NewClient(config.ResendAPIKey)

	return &notificationClient{
		config: config,
		client: client,
	}
}
