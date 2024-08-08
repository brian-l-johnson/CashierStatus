package models

type VerifyMacReq struct {
	Action string `json:"action"`
	Value  string `json:"value"`
	Mac    string `json:"mac"`
}
