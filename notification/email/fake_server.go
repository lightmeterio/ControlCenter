// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package email

import (
	"errors"
	"io"
	"net/mail"
	"time"

	smtp "github.com/emersion/go-smtp"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type FakeMailBackend struct {
	ExpectedUser     string
	ExpectedPassword string
	Messages         []*mail.Message
}

func (b *FakeMailBackend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &fakeSession{backend: b}, nil
}

type fakeSession struct {
	backend *FakeMailBackend
}

func (s *fakeSession) AuthPlain(username, password string) error {
	if username != s.backend.ExpectedUser || password != s.backend.ExpectedPassword {
		return errors.New("Invalid username or password")
	}

	return nil
}

func (s *fakeSession) Mail(from string, opts *smtp.MailOptions) error {
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
