package model

type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type WithdrawalsResponses []WithdrawalsResponse

type WithdrawalsResponse struct {
	OrderNumber string  `json:"order"`
	Sum         float64 `json:"sum"`
	DateCreated string  `json:"processed_at"`
}
