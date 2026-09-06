// Package mailer sends transactional email (currently just password
// resets) over SMTP. Gmail is the target, on port 587 with STARTTLS.
package mailer

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

// net/smtp's SendMail dials with no deadline, so a filtered port holds the
// connection open indefinitely.
const (
	dialTimeout = 10 * time.Second
	sendTimeout = 30 * time.Second
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
// the reset link instead of failing the request.
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

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(m.host, m.port), dialTimeout)
	if err != nil {
		return fmt.Errorf("dial smtp: %w", err)
	}
	defer conn.Close()

	// Covers a server that connects and then stalls mid-conversation.
	if err := conn.SetDeadline(time.Now().Add(sendTimeout)); err != nil {
		return fmt.Errorf("set smtp deadline: %w", err)
	}

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("smtp handshake: %w", err)
	}
	defer client.Close()

	// STARTTLS before Auth: PlainAuth refuses to send credentials over an
	// unencrypted connection, so the wrong order stops mail working.
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: m.host}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
	}

	if err := client.Auth(smtp.PlainAuth("", m.username, m.password, m.host)); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("smtp from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close message: %w", err)
	}

	return client.Quit()
}
