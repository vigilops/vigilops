package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"github.com/resend/resend-go/v3"
)

type resendClient struct {
	client    *resend.Client
	fromEmail string
}

func NewResendClient(apiKey, fromEmail string) Client {
	return &resendClient{
		client:    resend.NewClient(apiKey),
		fromEmail: fromEmail,
	}
}

func (r *resendClient) Send(templateFile, email string, data any) error {
	tmpl, err := template.ParseFS(FS, "templates/"+templateFile)
	if err != nil {
		return err
	}

	subject := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(subject, "subject", data); err != nil {
		return err
	}

	body := new(bytes.Buffer)
	if err := tmpl.ExecuteTemplate(body, "body", data); err != nil {
		return err
	}

	params := &resend.SendEmailRequest{
		From:    FromName + " <" + r.fromEmail + ">",
		To:      []string{email},
		Subject: subject.String(),
		Html:    body.String(),
	}

	var retryErr error
	for i := range maxRetries {
		_, retryErr = r.client.Emails.Send(params)
		if retryErr == nil {
			return nil
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}
	return fmt.Errorf("mailer: failed after %d attempts: %w", maxRetries, retryErr)
}
