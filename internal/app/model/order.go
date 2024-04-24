package model

type OrderStatus string

const (
	StatusNewOrder        OrderStatus = `NEW`
	StatusProcessingOrder OrderStatus = `PROCESSING`
	StatusInvalidOrder    OrderStatus = `INVALID`
	StatusProcessedOrder  OrderStatus = `PROCESSED`
)

type UploadedOrders []UploadedOrder

type UploadedOrder struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	Accrual    int    `json:"accrual"`
	UploadTime string `json:"uploaded_at"`
}
