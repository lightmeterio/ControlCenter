// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	_ "gitlab.com/lightmeter/controlcenter/auth/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/metadata"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
)

type UserData struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Registrar interface {
	Register(ctx context.Context, email, name, password string) (int64, error)
	Authenticate(ctx context.Context, email, password string) (bool, *UserData, error)
	HasAnyUser(ctx context.Context) (bool, error)
	GetUserDataByID(ctx context.Context, id int) (*UserData, error)
	GetFirstUser(ctx context.Context) (*UserData, error)
	ChangeUserInfo(ctx context.Context, oldEmail, newEmail, newName, newPassword string) error
}

type RegistrarWithSessionKeys interface {
	Registrar
	SessionKeys() [][]byte
}

type PlainAuthOptions struct {
	Email    string
	Name     string
	Password string
}

type Options struct {
	AllowMultipleUsers bool
	PlainAuthOptions   *PlainAuthOptions
}

type Auth struct {
	options  Options
	connPair *dbconn.PooledPair
	meta     *metadata.Handler
}

var (
	ErrGeneralAuthenticationError = errors.New("General Authentication Error")
	ErrUserAlreadyRegistred       = errors.New("User is Already Registred")
	ErrRegistrationDenied         = errors.New("Maximum Number of Users Already Registered")
)

func userIsAlreadyRegistred(tx *sql.Tx, email string) error {
	var count int

	if err := tx.QueryRow(`select count(*) from users where email = ?`, email).Scan(&count); err != nil {
		return errorutil.Wrap(err)
	}

	if count > 0 {
		log.Info().Msgf("Failed Attempt to register with the email %s again", email)

		// Here it's okay (privacy wise) and useful to inform that the user is already registred
		return errorutil.Wrap(ErrUserAlreadyRegistred)
	}

	return nil
}

const (
	invalidUserId = -1
)

func (r *Auth) Register(ctx context.Context, email, name, password string) (int64, error) {
	hasAnyUser, err := r.HasAnyUser(ctx)

	if err != nil {
		return invalidUserId, errorutil.Wrap(err)
	}

	if !r.options.AllowMultipleUsers && hasAnyUser {
		return invalidUserId, ErrRegistrationDenied
	}

	if err := validatePassword(email, name, password); err != nil {
		return invalidUserId, errorutil.Wrap(err)
	}

	if err := validateEmail(email); err != nil {
		return invalidUserId, errorutil.Wrap(err)
	}

	if err := validateName(name); err != nil {
		return invalidUserId, errorutil.Wrap(err)
	}

	id, err := registerInDb(ctx, r.connPair.RwConn, email, name, password)
	if err != nil {
		return invalidUserId, errorutil.Wrap(err)
	}

	return id, err
}

func registerInDb(ctx context.Context, db dbconn.RwConn, email, name, password string) (int64, error) {
	var id int64

	if err := db.Tx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if err := userIsAlreadyRegistred(tx, email); err != nil {
			return errorutil.Wrap(err)
		}

		result, err := tx.Exec(`insert into users(email, name, password) values(?, ?, lm_bcrypt_sum(?))`, email, name, password)

		if err != nil {
			return errorutil.Wrap(err, "executing user registration query")
		}

		id, err = result.LastInsertId()
		if err != nil {
			return errorutil.Wrap(err)
		}

		return nil
	}); err != nil {
		return invalidUserId, errorutil.Wrap(err)
	}

	log.Info().Msgf("Registering user %v with id %v", email, id)

	return id, nil
}

func (r *Auth) Authenticate(ctx context.Context, email, password string) (bool, *UserData, error) {
	d := UserData{}

	conn, release, err := r.connPair.RoConnPool.AcquireContext(ctx)
	if err != nil {
		return false, nil, errorutil.Wrap(err)
	}

	defer release()

	err = conn.
		QueryRowContext(ctx, "select rowid, email, name from users where email = ? and lm_bcrypt_compare(password, ?)", email, password).
		Scan(&d.Id, &d.Email, &d.Name)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil, nil
	}

	if err != nil {
		return false, nil, errorutil.Wrap(err)
	}

	return true, &d, nil
}

func (r *Auth) HasAnyUser(ctx context.Context) (bool, error) {
	var count int

	conn, release, err := r.connPair.RoConnPool.AcquireContext(ctx)
	if err != nil {
		return false, errorutil.Wrap(err)
	}

	defer release()

	if err := conn.QueryRowContext(ctx, "select count(*) from users").Scan(&count); err != nil {
		return false, errorutil.Wrap(err)
	}

	return count > 0, nil
}

var (
	ErrInvalidUserId = errors.New(`Invalid User ID`)
	ErrNoUser        = errors.New(`No registered user`)
)

func (r *Auth) GetUserDataByID(ctx context.Context, id int) (*UserData, error) {
	var userData UserData

	conn, release, err := r.connPair.RoConnPool.AcquireContext(ctx)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer release()

	err = conn.QueryRowContext(ctx, "select rowid, name, email from users where rowid = ?", id).Scan(&userData.Id, &userData.Name, &userData.Email)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidUserId
	}

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &userData, nil
}

func generateKeys() ([][]byte, error) {
	sessionKey := make([]byte, 32)

	n, err := io.ReadFull(rand.Reader, sessionKey)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	log.Info().Msgf("Generated Session key with %v bytes", n)

	return [][]byte{sessionKey}, nil
}

func (r *Auth) SessionKeys() [][]byte {
	keys := [][]byte{}

	ctx := context.Background()

	err := r.meta.Reader.RetrieveJson(ctx, "session_key", &keys)

	if err != nil && !errors.Is(err, metadata.ErrNoSuchKey) {
		errorutil.MustSucceed(err, "Obtaining session keys from database")
	}

	if len(keys) > 0 {
		return keys
	}

	keys, err = generateKeys()

	errorutil.MustSucceed(err, "Generating session keys")

	err = r.meta.Writer.StoreJson(ctx, "session_key", keys)

	errorutil.MustSucceed(err, "Generating session keys")

	return keys
}

func NewAuth(connPair *dbconn.PooledPair, options Options) (RegistrarWithSessionKeys, error) {
	m, err := metadata.NewHandler(connPair)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	a := &Auth{options: options, connPair: connPair, meta: m}

	if options.PlainAuthOptions == nil {
		return a, nil
	}

	authOptions := options.PlainAuthOptions

	hasAnyUsers, err := a.HasAnyUser(context.Background())
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	if hasAnyUsers {
		// if an user is already registred, resets its info to the one passed as option
		oldInfo, err := a.GetFirstUser(context.Background())
		if err != nil {
			return nil, errorutil.Wrap(err)
		}

		if err := a.ChangeUserInfo(context.Background(), oldInfo.Email, authOptions.Email, authOptions.Name, authOptions.Password); err != nil {
			return nil, errorutil.Wrap(err)
		}

		return a, nil
	}

	if _, err := a.Register(context.Background(),
		authOptions.Email, authOptions.Name,
		authOptions.Password); err != nil {
		return nil, errorutil.Wrap(err)
	}

	return a, nil
}

func nameForEmail(tx *sql.Tx, email string) (string, error) {
	var name string

	err := tx.QueryRow(`select name from users where email = ?`, email).Scan(&name)

	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrEmailAddressNotFound
	}

	if err != nil {
		return "", errorutil.Wrap(err)
	}

	return name, nil
}

func (r *Auth) GetFirstUser(ctx context.Context) (*UserData, error) {
	var userData UserData

	conn, release, err := r.connPair.RoConnPool.AcquireContext(ctx)
	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer release()

	err = conn.QueryRowContext(ctx, "select rowid, name, email from users order by rowid asc limit 1").Scan(&userData.Id, &userData.Name, &userData.Email)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoUser
	}

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &userData, nil
}

func queryMustChangeOneRecord(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) error {
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return errorutil.Wrap(err)
	}

	rowsAfftected, err := result.RowsAffected()
	if err != nil {
		return errorutil.Wrap(err)
	}

	if rowsAfftected != 1 {
		return ErrEmailAddressNotFound
	}

	return nil
}

func (r *Auth) ChangeUserInfo(ctx context.Context, oldEmail, newEmail, newName, newPassword string) error {
	return r.connPair.RwConn.Tx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if len(newName) > 0 {
			if err := queryMustChangeOneRecord(ctx, tx, `update users set name = ? where email = ?`, newName, oldEmail); err != nil {
				return errorutil.Wrap(err)
			}
		}

		if len(newPassword) > 0 {
			name, err := nameForEmail(tx, oldEmail)

			if err != nil {
				return errorutil.Wrap(err)
			}

			if err := validatePassword(oldEmail, name, newPassword); err != nil {
				return errorutil.Wrap(err)
			}

			if err = queryMustChangeOneRecord(ctx, tx, `update users set password = lm_bcrypt_sum(?) where email = ?`, newPassword, oldEmail); err != nil {
				return errorutil.Wrap(err)
			}
		}

		if len(newEmail) > 0 {
			if err := validateEmail(newEmail); err != nil {
				return errorutil.Wrap(err)
			}

			if err := queryMustChangeOneRecord(ctx, tx, `update users set email = ? where email = ?`, newEmail, oldEmail); err != nil {
				return errorutil.Wrap(err)
			}
		}

		return nil
	})
}
