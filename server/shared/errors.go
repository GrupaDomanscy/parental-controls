package shared

import "errors"

var ErrInvalidJsonPayload = errors.New("Invalid json payload")
var ErrInvalidEmail = errors.New("Invalid email")
var ErrInvalidCallbackUrl = errors.New("Invalid callback url")
var ErrUserWithGivenEmailAlreadyExists = errors.New("user with given email already exists")
var ErrInvalidOtat = errors.New("invalid one time access token")
