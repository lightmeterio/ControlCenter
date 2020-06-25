package httpauth

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/lightmeter/controlcenter/auth"
)

type fakeRegistrar struct {
	email                             string
	name                              string
	password                          string
	authenticated                     bool
	shouldFailToRegister              bool
	shouldFailToAuthenticate          bool
	authenticateYieldsError           bool
	shouldFailToCheckIfThereIsAnyUser bool
}

func (f *fakeRegistrar) Register(email, name, password string) error {
	if f.shouldFailToRegister {
		return errors.New("Weak Password")
	}

	f.authenticated = true
	f.name = name
	f.email = email
	f.password = password
	return nil
}

func (f *fakeRegistrar) HasAnyUser() (bool, error) {
	if f.shouldFailToCheckIfThereIsAnyUser {
		return false, errors.New("Some very severe error. Really")
	}

	return len(f.email) > 0, nil
}

func (f *fakeRegistrar) Authenticate(email, password string) (bool, auth.UserData, error) {
	if f.authenticateYieldsError {
		return false, auth.UserData{}, errors.New("Fail On Authentication")
	}

	if f.shouldFailToAuthenticate {
		return false, auth.UserData{}, nil
	}

	return email == f.email && password == f.password, auth.UserData{Name: f.name, Email: f.email}, nil
}

func (f *fakeRegistrar) CookieStore() sessions.Store {
	return sessions.NewCookieStore([]byte("SUPER_SECRET_KEYS"))
}

func TestHTTPAuth(t *testing.T) {
	Convey("HTTP Authentication", t, func() {
		failedAttempts := 0
		logoutAttempts := 0

		registrar := &fakeRegistrar{
			authenticated:                     false,
			shouldFailToRegister:              false,
			shouldFailToAuthenticate:          false,
			shouldFailToCheckIfThereIsAnyUser: false,
		}

		authenticator := NewAuthenticatorWithOptions(
			AuthHandlers{
				Unauthorized: func(w http.ResponseWriter, r *http.Request) {
					failedAttempts++
				},
				Public: func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("Public: " + r.URL.Path))
				},
				ShowLogin: func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("Login Page Content"))
				},
				Register: func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("Registration Page Content"))
				},
				LoginFailure: func(w http.ResponseWriter, r *http.Request) {
				},
				SecretArea: func(session SessionData, w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("Secret Area, dear " + session.Name))
				},
				Logout: func(session SessionData, w http.ResponseWriter, r *http.Request) {
					logoutAttempts++
				},
				ServerError: func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("Server Error"))
				},
			},
			registrar,
			[]string{"/public", "/visible"},
		)

		s := httptest.NewServer(authenticator)

		defer s.Close()

		buildCookieClient := func() *http.Client {
			jar, err := cookiejar.New(&cookiejar.Options{})
			So(err, ShouldBeNil)
			return &http.Client{Jar: jar}
		}

		Convey("Unauthenticated but registred user", func() {
			c := buildCookieClient()
			defer c.CloseIdleConnections()

			// register an user
			registrar.email = "user@example.com"
			registrar.name = "User"
			registrar.password = "123456"

			Convey("Redirects to the login page", func() {
				r, err := c.Get(s.URL)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
				So(failedAttempts, ShouldEqual, 1)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Login Page Content")
			})

			Convey("Error happens checking whether there is any user", func() {
				registrar.shouldFailToCheckIfThereIsAnyUser = true
				r, err := c.Get(s.URL)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Server Error")
			})

			Convey("Logout when user is not logged in goes to login page", func() {
				r, err := c.Get(s.URL + "/logout")
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
				So(failedAttempts, ShouldEqual, 1)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Login Page Content")
			})
		})

		Convey("Unauthenticated and unregistred user", func() {
			c := buildCookieClient()
			defer c.CloseIdleConnections()

			Convey("Logout when user is not logged in goes to registration page", func() {
				r, err := c.Get(s.URL + "/logout")
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
				So(failedAttempts, ShouldEqual, 1)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Registration Page Content")
			})

			Convey("Stay in the login page", func() {
				r, err := c.Get(s.URL + "/login")
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
				So(failedAttempts, ShouldEqual, 0)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Login Page Content")
			})

			Convey("Redirects to registration otherwise", func() {
				Convey("From main page", func() {
					r, err := c.Get(s.URL)
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)
					So(failedAttempts, ShouldEqual, 1)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Registration Page Content")
				})

				Convey("From some arbitrary page", func() {
					r, err := c.Get(s.URL + "/some/nested/resource/")
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)
					So(failedAttempts, ShouldEqual, 1)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Registration Page Content")
				})
			})

			Convey("Stay in the registration page", func() {
				r, err := c.Get(s.URL + "/register")
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
				So(failedAttempts, ShouldEqual, 0)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Registration Page Content")
			})

			Convey("Login will fail on invalid method", func() {
				req, err := http.NewRequest("DELETE", s.URL+"/login", nil)
				So(err, ShouldBeNil)
				r, err := c.Do(req)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
				So(failedAttempts, ShouldEqual, 0)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Server Error")
			})

			Convey("Login will fail on wrong request mime", func() {
				r, err := c.Post(s.URL+"/login", "application/json", nil)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
				So(failedAttempts, ShouldEqual, 0)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Server Error")
			})

			Convey("Login will fail on invalid request mime", func() {
				r, err := c.Post(s.URL+"/login", "ksajdhfk*I&^&*^87678  $$343", nil)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
				So(failedAttempts, ShouldEqual, 0)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Server Error")
			})

			Convey("Login will fail on invalid form data", func() {
				r, err := c.Post(s.URL+"/login", "application/x-www-form-urlencoded", strings.NewReader(`^^%`))
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
				So(failedAttempts, ShouldEqual, 0)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Server Error")
			})

			Convey("Login will fail due to some error with the authenticator", func() {
				registrar.authenticateYieldsError = true
				r, err := c.PostForm(s.URL+"/login", url.Values{"email": {"alice@example.com"}, "password": {"some_password"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
				So(failedAttempts, ShouldEqual, 0)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Server Error")
			})

			Convey("Login will fail as there is no registred user", func() {
				r, err := c.PostForm(s.URL+"/login", url.Values{"email": {"alice@example.com"}, "password": {"some_password"}})
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
				So(failedAttempts, ShouldEqual, 0)
				body, _ := ioutil.ReadAll(r.Body)
				response := struct{ Error string }{}
				So(json.Unmarshal(body, &response), ShouldBeNil)
				So(response.Error, ShouldEqual, "Invalid email address or password")
			})

			Convey("User registrations fails", func() {
				Convey("Invalid HTTP method", func() {
					req, err := http.NewRequest("DELETE", s.URL+"/register", nil)
					So(err, ShouldBeNil)
					r, err := c.Do(req)
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
					So(failedAttempts, ShouldEqual, 0)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Server Error")
				})

				Convey("Invalid form mime type", func() {
					r, err := c.Post(s.URL+"/register", "application/json", strings.NewReader(`{}`))
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
					So(failedAttempts, ShouldEqual, 0)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Server Error")
				})

				Convey("Invalid Form data", func() {
					r, err := c.Post(s.URL+"/register", "application/x-www-form-urlencoded", strings.NewReader(`^^%`))
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
					So(failedAttempts, ShouldEqual, 0)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Server Error")
				})

				Convey("No email and password provided", func() {
					r, err := c.PostForm(s.URL+"/register", url.Values{})
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
					So(failedAttempts, ShouldEqual, 0)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Server Error")
				})

				Convey("No email provided", func() {
					r, err := c.PostForm(s.URL+"/register", url.Values{
						"password": {"some_password"},
					})

					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
					So(failedAttempts, ShouldEqual, 0)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Server Error")
				})

				Convey("No password provided", func() {
					r, err := c.PostForm(s.URL+"/register", url.Values{
						"email": {"alice@example.com"},
					})

					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
					So(failedAttempts, ShouldEqual, 0)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Server Error")
				})

				Convey("Some validation makes the registring fail", func() {
					registrar.shouldFailToRegister = true

					r, err := c.PostForm(s.URL+"/register", url.Values{
						"email":    {"alice@example.com"},
						"password": {"poor password"},
					})

					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
					body, _ := ioutil.ReadAll(r.Body)

					response := struct {
						Error string
					}{}

					So(json.Unmarshal(body, &response), ShouldBeNil)

					So(response.Error, ShouldEqual, "Weak Password")
				})
			})

			Convey("User registrations succeeds", func() {
				r, err := c.PostForm(s.URL+"/register", url.Values{
					"email":    {"alice@example.com"},
					"name":     {"Alice"},
					"password": {"correcthorsebatterystable"},
				})

				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)

				body, _ := ioutil.ReadAll(r.Body)

				response := struct {
					Error string
				}{}

				So(json.Unmarshal(body, &response), ShouldBeNil)
				So(response.Error, ShouldEqual, "")

				Convey("After registred, the user is authenticated", func() {
					r, err := c.Get(s.URL) // go to main page
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)
					So(failedAttempts, ShouldEqual, 0)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Secret Area, dear Alice")
				})

				Convey("After registred, going to login page redirects to the main page", func() {
					r, err := c.Get(s.URL + "/login")
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)
					So(failedAttempts, ShouldEqual, 0)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Secret Area, dear Alice")
				})

				Convey("After registred, going to registration page redirects to the main page", func() {
					r, err := c.Get(s.URL + "/register")
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)
					So(failedAttempts, ShouldEqual, 0)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Secret Area, dear Alice")
				})

				Convey("User logs out, returning to the login page", func() {
					r, err := c.Get(s.URL + "/logout")
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)
					So(failedAttempts, ShouldEqual, 0)
					So(logoutAttempts, ShouldEqual, 1)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Login Page Content")

					Convey("User is unauthorized again", func() {
						Convey("From main page", func() {
							r, err := c.Get(s.URL)
							So(err, ShouldBeNil)
							So(r.StatusCode, ShouldEqual, http.StatusOK)
							So(failedAttempts, ShouldEqual, 1)
							body, _ := ioutil.ReadAll(r.Body)
							So(string(body), ShouldEqual, "Login Page Content")
						})

						Convey("From some arbitrary page", func() {
							r, err := c.Get(s.URL + "/some/nested/resource/")
							So(err, ShouldBeNil)
							So(r.StatusCode, ShouldEqual, http.StatusOK)
							So(failedAttempts, ShouldEqual, 1)
							body, _ := ioutil.ReadAll(r.Body)
							So(string(body), ShouldEqual, "Login Page Content")
						})
					})

					Convey("User can login again", func() {
						r, err := c.PostForm(s.URL+"/login", url.Values{"email": {"alice@example.com"}, "password": {"correcthorsebatterystable"}})
						So(err, ShouldBeNil)
						So(r.StatusCode, ShouldEqual, http.StatusOK)
						So(failedAttempts, ShouldEqual, 0)
						body, _ := ioutil.ReadAll(r.Body)
						response := struct{ Error string }{}
						So(json.Unmarshal(body, &response), ShouldBeNil)
						So(response.Error, ShouldEqual, "")
					})

					Convey("User can login again with posting more complex mime-type", func() {
						formData := url.Values{"email": {"alice@example.com"}, "password": {"correcthorsebatterystable"}}
						r, err := c.Post(s.URL+"/login", "application/x-www-form-urlencoded;charset=UTF-8", strings.NewReader(formData.Encode()))
						So(err, ShouldBeNil)
						So(r.StatusCode, ShouldEqual, http.StatusOK)
						So(failedAttempts, ShouldEqual, 0)
						body, _ := ioutil.ReadAll(r.Body)
						response := struct{ Error string }{}
						So(json.Unmarshal(body, &response), ShouldBeNil)
						So(response.Error, ShouldEqual, "")
					})
				})
			})
		})

		Convey("Simple HTTP Client with no cookies, using basic http authentication", func() {
			c := &http.Client{}
			defer c.CloseIdleConnections()

			Convey("Unregistred User fails to connect", func() {
				req, err := http.NewRequest("GET", s.URL+"/secret/area", nil)
				So(err, ShouldBeNil)
				req.SetBasicAuth("user@example.com", "123456")
				r, err := c.Do(req)
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
			})

			Convey("User Is registred", func() {
				registrar.email = "user@example.com"
				registrar.password = "654321"
				registrar.name = "Sakura"

				Convey("Auth fails due wrong credentials", func() {
					req, err := http.NewRequest("GET", s.URL+"/secret/area", nil)
					So(err, ShouldBeNil)
					req.SetBasicAuth("user@example.com", "wrong_password")
					r, err := c.Do(req)
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusUnauthorized)
				})

				Convey("Auth fails due internal error", func() {
					registrar.authenticateYieldsError = true
					req, err := http.NewRequest("GET", s.URL+"/secret/area", nil)
					So(err, ShouldBeNil)
					req.SetBasicAuth("user@example.com", "654321")
					r, err := c.Do(req)
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusInternalServerError)
				})

				Convey("Auth succeeds on correct credentials", func() {
					req, err := http.NewRequest("GET", s.URL+"/secret/area", nil)
					So(err, ShouldBeNil)
					req.SetBasicAuth("user@example.com", "654321")
					r, err := c.Do(req)
					So(err, ShouldBeNil)
					So(r.StatusCode, ShouldEqual, http.StatusOK)
					body, _ := ioutil.ReadAll(r.Body)
					So(string(body), ShouldEqual, "Secret Area, dear Sakura")
				})
			})
		})

		Convey("Accesses public paths", func() {
			Convey("With Cookies", func() {
				c := buildCookieClient()
				defer c.CloseIdleConnections()

				r, err := c.Get(s.URL + "/public/resource")
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
				So(failedAttempts, ShouldEqual, 0)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Public: /public/resource")
			})

			Convey("Without Cookies", func() {
				c := &http.Client{}
				defer c.CloseIdleConnections()

				r, err := c.Get(s.URL + "/visible")
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
				So(failedAttempts, ShouldEqual, 0)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Public: /visible")
			})

			Convey("Close prefix, but not matching, redirects to default page", func() {
				c := &http.Client{}
				defer c.CloseIdleConnections()

				r, err := c.Get(s.URL + "/publicaly_private/resource")
				So(err, ShouldBeNil)
				So(r.StatusCode, ShouldEqual, http.StatusOK)
				So(failedAttempts, ShouldEqual, 1)
				body, _ := ioutil.ReadAll(r.Body)
				So(string(body), ShouldEqual, "Registration Page Content")
			})
		})
	})
}
