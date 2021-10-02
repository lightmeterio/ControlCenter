// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package workspace

import (
	"context"
	"gitlab.com/lightmeter/controlcenter/auth"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
)

type PlainAuthOptions struct {
	Email    string
	Name     string
	Password string
}

type plainAuth struct {
	baseAuth auth.RegistrarWithSessionKeys
	options  PlainAuthOptions
}

func userDataFromPlainAuth(a *plainAuth) *auth.UserData {
	return &auth.UserData{Id: 1, Name: a.options.Name, Email: a.options.Email}
}

func (a *plainAuth) SessionKeys() [][]byte {
	return a.baseAuth.SessionKeys()
}

func (a *plainAuth) Register(ctx context.Context, email, name, password string) (int64, error) {
	return 1, nil
}

func (a *plainAuth) Authenticate(ctx context.Context, email, password string) (bool, *auth.UserData, error) {
	return email == a.options.Email && password == a.options.Password, userDataFromPlainAuth(a), nil
}

func (a *plainAuth) HasAnyUser(ctx context.Context) (bool, error) {
	return true, nil
}

func (a *plainAuth) GetUserDataByID(ctx context.Context, id int) (*auth.UserData, error) {
	return userDataFromPlainAuth(a), nil
}

func (a *plainAuth) GetFirstUser(ctx context.Context) (*auth.UserData, error) {
	return userDataFromPlainAuth(a), nil
}

func (a *plainAuth) ChangeUserInfo(ctx context.Context, oldEmail, newEmail, newName, newPassword string) error {
	return nil
}

func buildAuth(db *dbconn.PooledPair, authOptions auth.Options, options *PlainAuthOptions) (auth.RegistrarWithSessionKeys, error) {
	a, err := auth.NewAuth(db, authOptions)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	if options == nil {
		return a, nil
	}

	// wrap it!
	return &plainAuth{baseAuth: a, options: *options}, nil
}
