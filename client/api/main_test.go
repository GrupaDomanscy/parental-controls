package api

import "testing"

func TestRegister(t *testing.T) {
	t.Run("returns ErrUserError with ErrInvalidEmail if email is invalid", func(t *testing.T) {
		emailAddress := "some';][/.,invalidemail@localhost.local"
		callback := "http://localhost:8080/registration_otat_callback"

		Register()
	})

	t.Run("returns ErrUserError with ErrUserWithGivenEmailAlreadyExists if user with given email already exists", func(t *testing.T) {
		//TODO
		t.FailNow()
	})

	t.Run("returns bearer token if everything went ok", func(t *testing.T) {
		//TODO
		t.FailNow()
	})
}
