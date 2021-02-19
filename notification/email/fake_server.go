// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package email

import (
	"errors"
	smtp "github.com/emersion/go-smtp"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"net/mail"
	"time"
)

type FakeMailBackend struct {
	ExpectedUser     string
	ExpectedPassword string
	Messages         []*mail.Message
}

// Login handles a login command with username and password.
func (b *FakeMailBackend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	if username != b.ExpectedUser || password != b.ExpectedPassword {
		return nil, errors.New("Invalid username or password")
	}

	return &fakeSession{backend: b}, nil
}

// AnonymousLogin requires clients to authenticate using SMTP AUTH before sending emails
func (b *FakeMailBackend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return nil, smtp.ErrAuthRequired
}

type fakeSession struct {
	backend *FakeMailBackend
}

func (s *fakeSession) Mail(from string, opts smtp.MailOptions) error {
	return nil
}

func (s *fakeSession) Rcpt(to string) error {
	return nil
}

func (s *fakeSession) Data(r io.Reader) error {
	m, err := mail.ReadMessage(r)
	if err != nil {
		return errorutil.Wrap(err)
	}

	s.backend.Messages = append(s.backend.Messages, m)

	return nil
}

func (s *fakeSession) Reset() {}

func (s *fakeSession) Logout() error {
	return nil
}

func StartFakeServer(backend smtp.Backend, addr string) func() error {
	fakeServer := smtp.NewServer(backend)

	fakeServer.Addr = addr
	fakeServer.Domain = "localhost"
	fakeServer.WriteTimeout = 10 * time.Second
	fakeServer.ReadTimeout = 10 * time.Second
	fakeServer.MaxMessageBytes = 1024 * 1024
	fakeServer.MaxRecipients = 50
	fakeServer.AllowInsecureAuth = true

	go func() {
		if err := fakeServer.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	// wait for the smtp server to start
	time.Sleep(time.Millisecond * 700)

	return func() error {
		return fakeServer.Close()
	}
}
