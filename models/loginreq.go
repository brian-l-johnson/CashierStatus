package models

type LoginReq struct {
	User     string `json:"user"`
	Password string `json:"password"`
}
