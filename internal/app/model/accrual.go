package model

type AccrualResponse struct {
	Number  string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual,omitempty"`
}
