package shared

type AuthLoginRequestBody struct {
	Email    string `json:"email"`
	Callback string `json:"callback"`
}

type AuthStartRegistrationProcessRequestBody struct {
	Email    string `json:"email"`
	Callback string `json:"callback"`
}
