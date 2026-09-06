// Package mailer sends transactional email (currently just password
// resets) over SMTP. Gmail is the target - net/smtp's SendMail already
// upgrades to STARTTLS when the server offers it, which Gmail always does
// on port 587, so no extra TLS handling is needed here.
package mailer

import (
	"fmt"
	"net/smtp"
)

type Mailer struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func New(host, port, username, password, from string) *Mailer {
	return &Mailer{host: host, port: port, username: username, password: password, from: from}
}

// Configured is false when SMTP credentials are absent, so callers can log
// the reset link to the console in dev instead of failing the request -
// the same "degrade, don't break" pattern as the Gemini client.
func (m *Mailer) Configured() bool {
	return m.host != "" && m.username != "" && m.password != ""
}

func (m *Mailer) SendPasswordReset(to, resetURL string) error {
	subject := "Reset your FileForge password"
	body := fmt.Sprintf(
		"Someone requested a password reset for this email address.\r\n\r\n"+
			"Reset your password: %s\r\n\r\n"+
			"This link expires in 1 hour. If you didn't request this, you can ignore this email.\r\n",
		resetURL,
	)
	return m.send(to, subject, body)
}

func (m *Mailer) send(to, subject, body string) error {
	if !m.Configured() {
		return fmt.Errorf("mailer not configured")
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		m.from, to, subject, body)

	auth := smtp.PlainAuth("", m.username, m.password, m.host)
	addr := m.host + ":" + m.port
	return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
}
