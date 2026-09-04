// Package notify sends the simulated "your parcel was delivered" email to
// MailHog over plain SMTP (no auth, no TLS - it is a dev sink).
package notify

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

type Mailer struct {
	addr string
	from string
}

func NewMailer(addr, from string) *Mailer {
	return &Mailer{addr: addr, from: from}
}

// Sent describes a message that was handed to the SMTP server.
type Sent struct {
	To      string
	Subject string
	SentAt  time.Time
}

func (m *Mailer) SendDelivered(toName, toEmail, trackingNumber, destination string) (Sent, error) {
	subject := fmt.Sprintf("Parcel %s delivered", trackingNumber)
	body := strings.Join([]string{
		"To: " + toName + " <" + toEmail + ">",
		"From: ParcelPigeon <" + m.from + ">",
		"Subject: " + subject,
		"",
		fmt.Sprintf("Hi %s,", toName),
		"",
		fmt.Sprintf("Your parcel %s has been delivered to %s.", trackingNumber, destination),
		"",
		"Thanks for flying ParcelPigeon.",
	}, "\r\n")

	err := smtp.SendMail(m.addr, nil, m.from, []string{toEmail}, []byte(body))
	if err != nil {
		return Sent{}, err
	}
	return Sent{To: toEmail, Subject: subject, SentAt: time.Now().UTC()}, nil
}
