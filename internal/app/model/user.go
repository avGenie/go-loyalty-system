package model

type UserCredentialsRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
