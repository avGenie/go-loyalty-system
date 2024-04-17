package model

type UploadedOrders []UploadedOrder

type UploadedOrder struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	Accrual    int    `json:"accrual"`
	UploadTime string `json:"uploaded_at"`
}
