package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"net/textproto"

	"github.com/jordan-wright/email"
	"github.com/pkg/errors"
	"github.com/russross/blackfriday/v2"
)

type Config struct {
	SMTPHost  string   `toml:"smtp_host"`
	SMTPPort  int      `toml:"smtp_port"`
	Sender    string   `toml:"sender"`
	Password  string   `toml:"password"`
	Receivers []string `toml:"receivers"`
}

// Mail struct holds necessary data to send emails.
type Mail struct {
	usePlainText bool
	smtpAuth     smtp.Auth
	config       Config
}

// New returns a new instance of a Mail notification service.
func New(config Config) *Mail {
	m := &Mail{
		usePlainText: false,
		config:       config,
	}
	m.smtpAuth = smtp.PlainAuth("", config.Sender, config.Password, config.SMTPHost)
	return m
}

// BodyType is used to specify the format of the body.
type BodyType int

const (
	// PlainText is used to specify that the body is plain text.
	PlainText BodyType = iota
	// HTML is used to specify that the body is HTML.
	HTML
)

// AuthenticateSMTP authenticates you to send emails via smtp.
// Example values: "", "test@gmail.com", "password123", "smtp.gmail.com"
// For more information about smtp authentication, see here:
//
//	-> https://pkg.go.dev/net/smtp#PlainAuth
func (m *Mail) AuthenticateSMTP(identity, userName, password, host string) {
	m.smtpAuth = smtp.PlainAuth(identity, userName, password, host)
}

// AddReceivers takes email addresses and adds them to the internal address list. The Send method will send
// a given message to all those addresses.
func (m *Mail) AddReceivers(addresses ...string) {
	m.config.Receivers = append(m.config.Receivers, addresses...)
}

// BodyFormat can be used to specify the format of the body.
// Default BodyType is HTML.
func (m *Mail) BodyFormat(format BodyType) {
	switch format {
	case PlainText:
		m.usePlainText = true
	default:
		m.usePlainText = false
	}
}

func (m *Mail) newEmail(subject, message string, usePlainText bool) *email.Email {
	msg := &email.Email{
		To:      m.config.Receivers,
		From:    m.config.Sender,
		Subject: subject,
		Headers: textproto.MIMEHeader{},
	}

	if usePlainText {
		msg.Text = []byte(message)
	} else {
		msg.HTML = []byte(message)
	}
	return msg
}

func (m Mail) send(ctx context.Context, msg *email.Email) error {
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		host := fmt.Sprintf("%s:%d", m.config.SMTPHost, m.config.SMTPPort)
		if m.config.SMTPPort == 25 {
			err = msg.Send(host, m.smtpAuth)
		} else {
			tlsconfig := &tls.Config{
				ServerName: m.config.SMTPHost,
				MinVersion: tls.VersionTLS12,
			}
			err = msg.SendWithTLS(host, m.smtpAuth, tlsconfig)
		}
		if err != nil {
			err = errors.Wrap(err, "failed to send mail")
		}
	}
	return err
}

// Send takes a message subject and a message body and sends them to all previously set chats. Message body supports
// html as markup language.
func (m Mail) Send(ctx context.Context, subject, message string) error {
	msg := m.newEmail(subject, message, m.usePlainText)
	return m.send(ctx, msg)
}

func (m Mail) SendMarkdown(ctx context.Context, subject, message string) error {
	html := blackfriday.Run([]byte(message))
	msg := m.newEmail(subject, string(html), false)
	return m.send(ctx, msg)
}
