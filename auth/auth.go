package auth

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util"
	"io"
	"log"
	"path"
)

type UserData struct {
	Id    int
	Name  string
	Email string
}

type Registrar interface {
	Register(email, name, password string) error
	Authenticate(email, password string) (bool, UserData, error)
	HasAnyUser() (bool, error)
}

type Options struct {
	AllowMultipleUsers bool
}

type Auth struct {
	options  Options
	connPair dbconn.ConnPair
}

var (
	ErrGeneralAuthenticationError = errors.New("General Authentication Error")
	ErrUserAlreadyRegistred       = errors.New("User is Already Registred")
	ErrRegistrationDenied         = errors.New("Maximum Number of Users Already Registered")
)

func userIsAlreadyRegistred(tx *sql.Tx, email string) error {
	var count int

	if err := tx.QueryRow(`select count(*) from users where email = ?`, email).Scan(&count); err != nil {
		return util.WrapError(err)
	}

	if count > 0 {
		log.Println("Failed Attempt to register with the email", email, "again")

		// Here it's okay (privacy wise) and useful to inform that the user is already registred
		return util.WrapError(ErrUserAlreadyRegistred)
	}

	return nil
}

func (r *Auth) Register(email, name, password string) error {
	hasAnyUser, err := r.HasAnyUser()

	if err != nil {
		return util.WrapError(err)
	}

	if !r.options.AllowMultipleUsers && hasAnyUser {
		return ErrRegistrationDenied
	}

	if err := validatePassword(email, name, password); err != nil {
		return util.WrapError(err)
	}

	if err := validateEmail(email); err != nil {
		return util.WrapError(err)
	}

	if err := validateName(name); err != nil {
		return util.WrapError(err)
	}

	if err := registerInDb(r.connPair.RwConn, email, name, password); err != nil {
		return util.WrapError(err)
	}

	return nil
}

func registerInDb(db *sql.DB, email, name, password string) error {
	tx, err := db.Begin()

	if err != nil {
		return util.WrapError(err)
	}

	defer func() {
		if err != nil {
			util.MustSucceed(tx.Rollback(), "Rolling back user registration transaction")
		}
	}()

	err = userIsAlreadyRegistred(tx, email)

	if err != nil {
		return util.WrapError(err)
	}

	result, err := tx.Exec(`insert into users(email, name, password) values(?, ?, lm_bcrypt_sum(?))`, email, name, password)

	if err != nil {
		return util.WrapError(err, "executing user registration query")
	}

	id, err := result.LastInsertId()

	if err != nil {
		return util.WrapError(err)
	}

	err = tx.Commit()

	if err != nil {
		return util.WrapError(err)
	}

	log.Println("Registering user", email, "with id", id)

	return nil
}

func (r *Auth) Authenticate(email, password string) (bool, UserData, error) {
	d := UserData{}

	err := r.connPair.RoConn.
		QueryRow("select rowid, email, name from users where email = ? and lm_bcrypt_compare(password, ?)", email, password).
		Scan(&d.Id, &d.Email, &d.Name)

	if errors.Is(err, sql.ErrNoRows) {
		return false, UserData{}, nil
	}

	if err != nil {
		return false, UserData{}, util.WrapError(err)
	}

	return true, d, nil
}

func (r *Auth) HasAnyUser() (bool, error) {
	var count int

	if err := r.connPair.RoConn.QueryRow("select count(*) from users").Scan(&count); err != nil {
		return false, util.WrapError(err)
	}

	return count > 0, nil
}

// NOTE: For some reason, rowserrcheck is not able to see that q.Err() is being called,
// so we disable the check here until the linter is fixed or someone finds the bug in this
// code.
//nolint:rowserrcheck
func tryToObtainExistingKeys(tx *sql.Tx) ([][]byte, error) {
	q, err := tx.Query(`select value from meta where key = ?`, "session_key")

	if err != nil {
		return nil, util.WrapError(err)
	}

	defer func() {
		util.MustSucceed(q.Close(), "")
	}()

	var sessionKeys [][]byte = nil

	for q.Next() {
		sessionKey := make([]byte, 32)

		if err := q.Scan(&sessionKey); err != nil {
			log.Println("Error reading session key from database:", err)
			return nil, util.WrapError(err)
		}

		sessionKeys = append(sessionKeys, sessionKey)
	}

	if err := q.Err(); err != nil {
		log.Println("Error on query for session key:", err)
		return nil, util.WrapError(err)
	}

	return sessionKeys, nil
}

func generateKeys(tx *sql.Tx) ([][]byte, error) {
	sessionKey := make([]byte, 32)

	n, err := io.ReadFull(rand.Reader, sessionKey)

	if err != nil {
		return nil, util.WrapError(err)
	}

	log.Println("Generated Session key with ", n, "bytes")

	_, err = tx.Exec(`insert into meta(key, value) values("session_key", ?)`, sessionKey)

	if err != nil {
		return nil, util.WrapError(err)
	}

	return [][]byte{sessionKey}, nil

}

func (r *Auth) SessionKeys() [][]byte {
	_, err := r.connPair.RwConn.Exec(`create table if not exists meta(
		key string,
		value blob
	)`)

	util.MustSucceed(err, "Ensuring auth meta table exists")

	tx, err := r.connPair.RwConn.Begin()

	util.MustSucceed(err, "Starting transaction on session key management")

	defer func() {
		if err == nil {
			util.MustSucceed(tx.Commit(), "")
		}
	}()

	existingKeys, err := tryToObtainExistingKeys(tx)

	util.MustSucceed(err, "Obtaining session keys from database")

	if existingKeys != nil {
		return existingKeys
	}

	keys, err := generateKeys(tx)

	util.MustSucceed(err, "Generating session keys")

	return keys
}

func setupWriterConnection(conn *sql.DB) error {
	if _, err := conn.Exec(`create table if not exists users(
		email string,
		name string,
		password blob
	)`); err != nil {
		return util.WrapError(err)
	}

	return nil
}

func NewAuth(dirname string, options Options) (*Auth, error) {
	connPair, err := dbconn.NewConnPair(path.Join(dirname, "auth.db"))

	if err != nil {
		return nil, util.WrapError(err)
	}

	defer func() {
		if err != nil {
			util.MustSucceed(connPair.Close(), "Closing DB connection on error")
		}
	}()

	err = setupWriterConnection(connPair.RwConn)

	if err != nil {
		return nil, util.WrapError(err)
	}

	return &Auth{options: options, connPair: connPair}, nil
}

func (r *Auth) Close() error {
	return r.connPair.Close()
}

func nameForEmail(tx *sql.Tx, email string) (string, error) {
	var name string

	err := tx.QueryRow(`select name from users where email = ?`, email).Scan(&name)

	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrEmailAddressNotFound
	}

	if err != nil {
		return "", util.WrapError(err)
	}

	return name, nil
}

func updatePassword(tx *sql.Tx, email, password string) error {
	_, err := tx.Exec(`update users set password = lm_bcrypt_sum(?) where email = ?`, password, email)

	if err != nil {
		return util.WrapError(err, "executing password reset query")
	}

	if err != nil {
		return util.WrapError(err)
	}

	return nil
}

func (r *Auth) ChangePassword(email, password string) error {
	tx, err := r.connPair.RwConn.Begin()

	if err != nil {
		return util.WrapError(err)
	}

	defer func() {
		if err != nil {
			util.MustSucceed(tx.Rollback(), "Rolling back attempt to change user password")
		}
	}()

	name, err := nameForEmail(tx, email)

	if err != nil {
		return util.WrapError(err)
	}

	if err := validatePassword(email, name, password); err != nil {
		return util.WrapError(err)
	}

	if err := updatePassword(tx, email, password); err != nil {
		return util.WrapError(err)
	}

	err = tx.Commit()

	if err != nil {
		return util.WrapError(err)
	}

	return nil
}
