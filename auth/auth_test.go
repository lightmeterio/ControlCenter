// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package auth

import (
	"context"
	"errors"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3"
	"gitlab.com/lightmeter/controlcenter/lmsqlite3/dbconn"
	"gitlab.com/lightmeter/controlcenter/util/testutil"
	"testing"
)

func init() {
	lmsqlite3.Initialize(lmsqlite3.Options{})
}

var (
	dummyContext = context.Background()
)

func TestSessionKey(t *testing.T) {
	Convey("Test Session Key", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, "auth")
		defer closeConn()

		var generatedKey, recoveredKey [][]byte

		// NOTE: for now we are generating only one key, but
		// generating multiple ones is desirable
		{
			auth, _ := NewAuth(conn, Options{})
			generatedKey = auth.SessionKeys()
			So(generatedKey, ShouldNotBeNil)
			So(len(generatedKey), ShouldEqual, 1)
			So(generatedKey[0], ShouldNotBeNil)
		}

		{
			auth, _ := NewAuth(conn, Options{})
			recoveredKey = auth.SessionKeys()
		}

		So(recoveredKey, ShouldResemble, generatedKey)
	})
}

func TestAuth(t *testing.T) {
	strongPassword := `ghjzfpailduifiapdq9um6ysuubvtjywAqbnadq+aUerxrqhfp`

	Convey("Test Auth", t, func() {
		conn, closeConn := testutil.TempDBConnectionMigrated(t, "auth")
		defer closeConn()

		auth, err := NewAuth(conn, Options{})
		So(err, ShouldBeNil)
		So(auth, ShouldNotBeNil)

		Convey("No user is initially registered", func() {
			ok, err := auth.HasAnyUser(dummyContext)
			So(err, ShouldBeNil)
			So(ok, ShouldBeFalse)

			Convey("GetFirstUser returns no user", func() {
				_, err = auth.GetFirstUser(dummyContext)
				So(err, ShouldEqual, ErrNoUser)
			})

			Convey("Login fails", func() {
				ok, _, err := auth.Authenticate(dummyContext, "user@example.com", "password")
				So(ok, ShouldBeFalse)
				So(err, ShouldBeNil)
			})
		})

		Convey("Register User Fails", func() {
			Convey("Empty password", func() {
				_, err := auth.Register(dummyContext, "user@example.com", "Name Surname", "")
				So(errors.Is(err, ErrWeakPassword), ShouldBeTrue)
			})

			Convey("Password is equal to email", func() {
				_, err := auth.Register(dummyContext, "user@example.com", "Name Surname", "user@example.com")
				So(errors.Is(err, ErrWeakPassword), ShouldBeTrue)
			})

			Convey("Password is equal to name", func() {
				_, err := auth.Register(dummyContext, "user@example.com", strongPassword, strongPassword)
				So(errors.Is(err, ErrWeakPassword), ShouldBeTrue)
			})

			Convey("Invalid email", func() {
				_, err := auth.Register(dummyContext, "not@an@email.com", "Name Surname", strongPassword)
				So(errors.Is(err, ErrInvalidEmail), ShouldBeTrue)
			})

			Convey("Dictionary password", func() {
				_, err := auth.Register(dummyContext, "user@email.com", "Name Surname", "ElvisForever")
				So(errors.Is(err, ErrWeakPassword), ShouldBeTrue)
			})

			Convey("Empty Name", func() {
				_, err := auth.Register(dummyContext, "user@email.com", "   ", strongPassword)
				So(errors.Is(err, ErrInvalidName), ShouldBeTrue)
			})

			Convey("Multiple users is forbidden", func() {
				// register one user, forbidding any others
				_, err := auth.Register(dummyContext, "user@email.com", "Valid Name", strongPassword)
				So(err, ShouldBeNil)

				_, err = auth.Register(dummyContext, "another.user@email.com", "Another User", strongPassword)
				So(errors.Is(err, ErrRegistrationDenied), ShouldBeTrue)
			})
		})

		Convey("Register Multiple Users", func() {
			auth, err := NewAuth(conn, Options{AllowMultipleUsers: true})
			So(err, ShouldBeNil)

			user1Passwd := `ymzlxzmojdnQ3revu/s2jnqbFydoqw`
			user2Passwd := `yp9nr1yog|cWzjDftgspdgkntkbjig`

			_, err = auth.Register(dummyContext, "user.one@example.com", "User One", user1Passwd)
			So(err, ShouldBeNil)
			_, err = auth.Register(dummyContext, "user.two@example.com", "User Two", user2Passwd)
			So(err, ShouldBeNil)

			Convey("Passwords do not mix", func() {
				{
					ok, _, err := auth.Authenticate(dummyContext, "user.one@example.com", user2Passwd)
					So(err, ShouldBeNil)
					So(ok, ShouldBeFalse)
				}

				{
					ok, _, err := auth.Authenticate(dummyContext, "user.two@example.com", user1Passwd)
					So(err, ShouldBeNil)
					So(ok, ShouldBeFalse)
				}

				{
					ok, _, err := auth.Authenticate(dummyContext, "user.two@example.com", user2Passwd)
					So(err, ShouldBeNil)
					So(ok, ShouldBeTrue)
				}
			})

			Convey("Double registration of the same user fails", func() {
				// register one user, forbidding any others
				_, err := auth.Register(dummyContext, "user@email.com", "Valid Name", strongPassword)
				So(err, ShouldBeNil)

				_, err = auth.Register(dummyContext, "user@email.com", "Another Valid User", `67567567HGFHGFHGhgfghfhg***&*`)
				So(errors.Is(err, ErrUserAlreadyRegistred), ShouldBeTrue)
			})

			Convey("GetFirstUser returns the user registered first", func() {
				firstUser, err := auth.GetFirstUser(dummyContext)
				So(err, ShouldBeNil)
				So(firstUser.Id, ShouldEqual, 1)
				So(firstUser.Email, ShouldEqual, "user.one@example.com")
				So(firstUser.Name, ShouldEqual, "User One")
			})
		})

		Convey("Register User", func() {
			_, err := auth.Register(dummyContext, "user@example.com", "Name Surname", strongPassword)
			So(err, ShouldBeNil)
			ok, err := auth.HasAnyUser(dummyContext)
			So(err, ShouldBeNil)
			So(ok, ShouldBeTrue)

			Convey("Registering the same user again fails", func() {
				_, err := auth.Register(dummyContext, "user@example.com", "Another Surname", strongPassword)
				So(err, ShouldNotBeNil)
			})

			Convey("Login fails with wrong user", func() {
				ok, _, err := auth.Authenticate(dummyContext, "wrong_user@example.com", strongPassword)
				So(ok, ShouldBeFalse)
				So(err, ShouldBeNil)
			})

			Convey("Login fails with wrong password", func() {
				ok, _, err := auth.Authenticate(dummyContext, "user@example.com", "654321")
				So(ok, ShouldBeFalse)
				So(err, ShouldBeNil)
			})

			Convey("Login succeeds", func() {
				ok, userData, err := auth.Authenticate(dummyContext, "user@example.com", strongPassword)
				So(ok, ShouldBeTrue)
				So(err, ShouldBeNil)
				So(userData.Id, ShouldEqual, 1)
				So(userData.Email, ShouldEqual, "user@example.com")
				So(userData.Name, ShouldEqual, "Name Surname")
			})

			Convey("User Data by ID", func() {
				Convey("Invalid ID", func() {
					_, err := auth.GetUserDataByID(dummyContext, 42)
					So(errors.Is(err, ErrInvalidUserId), ShouldBeTrue)
				})
			})

			Convey("Valid ID", func() {
				userData, err := auth.GetUserDataByID(dummyContext, 1)
				So(err, ShouldBeNil)
				So(userData.Id, ShouldEqual, 1)
				So(userData.Email, ShouldEqual, "user@example.com")
				So(userData.Name, ShouldEqual, "Name Surname")
			})

			Convey("GetFirstUser succeeds", func() {
				firstUser, err := auth.GetFirstUser(dummyContext)
				So(err, ShouldBeNil)
				So(firstUser.Id, ShouldEqual, 1)
				So(firstUser.Email, ShouldEqual, "user@example.com")
				So(firstUser.Name, ShouldEqual, "Name Surname")
			})
		})
	})
}

const originalTestPassword = `(1Yow@byU]>`

func tempWorkspaceWithUserSetup(t *testing.T) (*dbconn.PooledPair, func()) {
	conn, closeConn := testutil.TempDBConnectionMigrated(t, "auth")

	auth, err := NewAuth(conn, Options{})
	So(err, ShouldBeNil)

	_, err = auth.Register(dummyContext, "email@example.com", `Nora`, originalTestPassword)
	So(err, ShouldBeNil)

	return conn, closeConn
}

func TestResetPassword(t *testing.T) {
	Convey("Reset Password", t, func() {
		conn, closeConn := tempWorkspaceWithUserSetup(t)
		defer closeConn()

		auth, err := NewAuth(conn, Options{})
		So(err, ShouldBeNil)

		Convey("Fails", func() {
			Convey("Invalid user", func() {
				So(errors.Is(auth.ChangeUserInfo(dummyContext, "invalid.user@example.com", ``, ``, `kjhjk^^776767&&&$123456`), ErrEmailAddressNotFound), ShouldBeTrue)
			})

			Convey("Too weak", func() {
				So(errors.Is(auth.ChangeUserInfo(dummyContext, "email@example.com", ``, ``, `123456`), ErrWeakPassword), ShouldBeTrue)
			})

			Convey("Equals email", func() {
				So(errors.Is(auth.ChangeUserInfo(dummyContext, "email@example.com", ``, ``, `email@example.com`), ErrWeakPassword), ShouldBeTrue)
			})

			Convey("Equals Name", func() {
				So(errors.Is(auth.ChangeUserInfo(dummyContext, "email@example.com", ``, ``, `Nora`), ErrWeakPassword), ShouldBeTrue)
			})
		})

		Convey("Succeeds", func() {
			So(auth.ChangeUserInfo(dummyContext, "email@example.com", ``, ``, `**^NeuEp4ssd:?&`), ShouldBeNil)

			ok, u, err := auth.Authenticate(dummyContext, "email@example.com", `**^NeuEp4ssd:?&`)

			So(err, ShouldBeNil)
			So(ok, ShouldBeTrue)
			So(u.Id, ShouldEqual, 1)
			So(u.Email, ShouldEqual, "email@example.com")
		})
	})
}

func TestChangeUserInfo(t *testing.T) {
	Convey("Change User Info", t, func() {
		conn, closeConn := tempWorkspaceWithUserSetup(t)
		defer closeConn()

		auth, err := NewAuth(conn, Options{})
		So(err, ShouldBeNil)

		Convey("Invalid user", func() {
			So(errors.Is(auth.ChangeUserInfo(dummyContext, "invalid.user@example.com", "new.email@example.com", "New Name", ``), ErrEmailAddressNotFound), ShouldBeTrue)
		})

		Convey("Invalid new e-mail", func() {
			So(errors.Is(auth.ChangeUserInfo(dummyContext, "email@example.com", "this-is-not-an-email-address...", "New Name", ``), ErrInvalidEmail), ShouldBeTrue)
		})

		Convey("Succeeds changing e-mail and name", func() {
			So(auth.ChangeUserInfo(dummyContext, "email@example.com", "new.email@example.com", "New Name", ``), ShouldBeNil)
			ok, u, err := auth.Authenticate(dummyContext, "new.email@example.com", originalTestPassword)
			So(err, ShouldBeNil)
			So(ok, ShouldBeTrue)
			So(u.Id, ShouldEqual, 1)
			So(u.Email, ShouldEqual, "new.email@example.com")
			So(u.Name, ShouldEqual, "New Name")
		})

		Convey("Succeeds changing e-mail only, leaving name intact", func() {
			So(auth.ChangeUserInfo(dummyContext, "email@example.com", "new.email@example.com", "", ``), ShouldBeNil)
			ok, u, err := auth.Authenticate(dummyContext, "new.email@example.com", originalTestPassword)
			So(err, ShouldBeNil)
			So(ok, ShouldBeTrue)
			So(u.Id, ShouldEqual, 1)
			So(u.Email, ShouldEqual, "new.email@example.com")
			So(u.Name, ShouldEqual, "Nora")
		})

		Convey("Succeeds changing name only, leaving e-mail intact", func() {
			So(auth.ChangeUserInfo(dummyContext, "email@example.com", "", "Alice", ``), ShouldBeNil)
			ok, u, err := auth.Authenticate(dummyContext, "email@example.com", originalTestPassword)
			So(err, ShouldBeNil)
			So(ok, ShouldBeTrue)
			So(u.Id, ShouldEqual, 1)
			So(u.Email, ShouldEqual, "email@example.com")
			So(u.Name, ShouldEqual, "Alice")
		})
	})
}
