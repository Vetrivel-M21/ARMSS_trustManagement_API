// Package mail sends outbound email through the GoDaddy-hosted mailbox
// (noreply@arminfo.in) via implicit-TLS SMTP (port 465). Generic on purpose —
// any feature needing to send email (currently just the installer OTP gate)
// should call this instead of talking to net/smtp directly.
package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"

	"trust-management/backend/internal/config"
)

type Service struct {
	cfg *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

// Send dials with TLS from the start of the connection — GoDaddy's port 465
// is implicit TLS (SMTPS), not STARTTLS, so the stdlib's smtp.SendMail
// helper (which only speaks STARTTLS over a plaintext-first connection)
// doesn't work here.
func (s *Service) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)

	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.cfg.SMTPHost})
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", s.cfg.SMTPUsername, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}

	if err := client.Mail(s.cfg.SMTPFromAddress); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		s.cfg.SMTPFromAddress, to, subject, body)
	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return client.Quit()
}
