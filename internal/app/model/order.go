package model

type AccrualOrderStatus string

const (
	StatusRegisteredAccrual AccrualOrderStatus = `REGISTERED`
	StatusInvalidAccrual    AccrualOrderStatus = `INVALID`
	StatusProcessingAccrual AccrualOrderStatus = `PROCESSING`
	StatusProcessedAccrual  AccrualOrderStatus = `PROCESSED`
)

type UploadedOrders []UploadedOrder

type UploadedOrder struct {
	Number     string  `json:"number"`
	Status     string  `json:"status"`
	Accrual    float64 `json:"accrual"`
	UploadTime string  `json:"uploaded_at"`
}
