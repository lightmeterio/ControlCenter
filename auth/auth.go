package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"github.com/rs/zerolog/log"
	_ "gitlab.com/lightmeter/controlcenter/auth/migrations"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/migrator"
	"gitlab.com/lightmeter/controlcenter/meta"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"path"
)

type UserData struct {
	Id    int
	Name  string
	Email string
}

type Registrar interface {
	Register(ctx context.Context, email, name, password string) (int64, error)
	Authenticate(ctx context.Context, email, password string) (bool, UserData, error)
	HasAnyUser(ctx context.Context) (bool, error)
	GetUserDataByID(ctx context.Context, id int) (*UserData, error)
}

type Options struct {
	AllowMultipleUsers bool
}

type Auth struct {
	options  Options
	connPair dbconn.ConnPair
	meta     *meta.Handler
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

func (r *Auth) Register(ctx context.Context, email, name, password string) (int64, error) {
	hasAnyUser, err := r.HasAnyUser(ctx)

	if err != nil {
		return -1, errorutil.Wrap(err)
	}

	if !r.options.AllowMultipleUsers && hasAnyUser {
		return -1, ErrRegistrationDenied
	}

	if err := validatePassword(email, name, password); err != nil {
		return -1, errorutil.Wrap(err)
	}

	if err := validateEmail(email); err != nil {
		return -1, errorutil.Wrap(err)
	}

	if err := validateName(name); err != nil {
		return -1, errorutil.Wrap(err)
	}

	id, err := registerInDb(ctx, r.connPair.RwConn, email, name, password)
	if err != nil {
		return -1, errorutil.Wrap(err)
	}

	return id, err
}

func registerInDb(ctx context.Context, db dbconn.RwConn, email, name, password string) (int64, error) {
	tx, err := db.BeginTx(ctx, nil)

	if err != nil {
		return -1, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(tx.Rollback(), "Rolling back user registration transaction")
		}
	}()

	err = userIsAlreadyRegistred(tx, email)

	if err != nil {
		return -1, errorutil.Wrap(err)
	}

	result, err := tx.Exec(`insert into users(email, name, password) values(?, ?, lm_bcrypt_sum(?))`, email, name, password)

	if err != nil {
		return -1, errorutil.Wrap(err, "executing user registration query")
	}

	id, err := result.LastInsertId()

	if err != nil {
		return -1, errorutil.Wrap(err)
	}

	err = tx.Commit()

	if err != nil {
		return -1, errorutil.Wrap(err)
	}

	log.Info().Msgf("Registering user %v with id %v", email, id)

	return id, nil
}

func (r *Auth) Authenticate(ctx context.Context, email, password string) (bool, UserData, error) {
	d := UserData{}

	err := r.connPair.RoConn.
		QueryRowContext(ctx, "select rowid, email, name from users where email = ? and lm_bcrypt_compare(password, ?)", email, password).
		Scan(&d.Id, &d.Email, &d.Name)

	if errors.Is(err, sql.ErrNoRows) {
		return false, UserData{}, nil
	}

	if err != nil {
		return false, UserData{}, errorutil.Wrap(err)
	}

	return true, d, nil
}

func (r *Auth) HasAnyUser(ctx context.Context) (bool, error) {
	var count int

	if err := r.connPair.RoConn.QueryRowContext(ctx, "select count(*) from users").Scan(&count); err != nil {
		return false, errorutil.Wrap(err)
	}

	return count > 0, nil
}

func (r *Auth) GetUserDataByID(ctx context.Context, id int) (*UserData, error) {
	var userData UserData

	if err := r.connPair.RoConn.QueryRowContext(ctx, "select rowid, name, email from users where rowid = ?", id).Scan(&userData.Id, &userData.Name, &userData.Email); err != nil {
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

	if err != nil && !errors.Is(err, meta.ErrNoSuchKey) {
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

func NewAuth(dirname string, options Options) (*Auth, error) {
	connPair, err := dbconn.NewConnPair(path.Join(dirname, "auth.db"))

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	if err := migrator.Run(connPair.RwConn.DB, "auth"); err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(connPair.Close(), "Closing DB connection on error")
		}
	}()

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	m, err := meta.NewHandler(connPair, "auth")

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	return &Auth{options: options, connPair: connPair, meta: m}, nil
}

func (r *Auth) Close() error {
	return r.meta.Close()
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

func updatePassword(tx *sql.Tx, email, password string) error {
	_, err := tx.Exec(`update users set password = lm_bcrypt_sum(?) where email = ?`, password, email)

	if err != nil {
		return errorutil.Wrap(err, "executing password reset query")
	}

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}

func (r *Auth) ChangePassword(ctx context.Context, email, password string) error {
	tx, err := r.connPair.RwConn.BeginTx(ctx, nil)

	if err != nil {
		return errorutil.Wrap(err)
	}

	defer func() {
		if err != nil {
			errorutil.MustSucceed(tx.Rollback(), "Rolling back attempt to change user password")
		}
	}()

	name, err := nameForEmail(tx, email)

	if err != nil {
		return errorutil.Wrap(err)
	}

	if err := validatePassword(email, name, password); err != nil {
		return errorutil.Wrap(err)
	}

	if err := updatePassword(tx, email, password); err != nil {
		return errorutil.Wrap(err)
	}

	err = tx.Commit()

	if err != nil {
		return errorutil.Wrap(err)
	}

	return nil
}
