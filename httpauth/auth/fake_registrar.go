// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"context"
	"errors"
	"github.com/gorilla/sessions"
	"gitlab.com/lightmeter/controlcenter/auth"
)

type FakeRegistrar struct {
	SessionKey                        []byte
	Email                             string
	Name                              string
	Password                          string
	Authenticated                     bool
	ShouldFailToRegister              bool
	ShouldFailToAuthenticate          bool
	AuthenticateYieldsError           bool
	ShouldFailToCheckIfThereIsAnyUser bool
}

func (f *FakeRegistrar) Register(ctx context.Context, email, name, password string) (int64, error) {
	if f.ShouldFailToRegister {
		return -1, errors.New("Weak Password")
	}

	f.Authenticated = true
	f.Name = name
	f.Email = email
	f.Password = password

	return 1, nil
}

func (f *FakeRegistrar) HasAnyUser(ctx context.Context) (bool, error) {
	if f.ShouldFailToCheckIfThereIsAnyUser {
		return false, errors.New("Some very severe error. Really")
	}

	return len(f.Email) > 0, nil
}

func (f *FakeRegistrar) GetUserDataByID(ctx context.Context, id int) (*auth.UserData, error) {
	return &auth.UserData{Id: 1, Name: "Donutloop", Email: "example@test.de"}, nil
}

func (f *FakeRegistrar) Authenticate(ctx context.Context, email, password string) (bool, auth.UserData, error) {
	if f.AuthenticateYieldsError {
		return false, auth.UserData{}, errors.New("Fail On Authentication")
	}

	if f.ShouldFailToAuthenticate {
		return false, auth.UserData{}, nil
	}

	return email == f.Email && password == f.Password, auth.UserData{Name: f.Name, Email: f.Email}, nil
}

func (f *FakeRegistrar) CookieStore() sessions.Store {
	return sessions.NewCookieStore(f.SessionKey)
}

func (f *FakeRegistrar) SessionKeys() [][]byte {
	return [][]byte{f.SessionKey}
}
