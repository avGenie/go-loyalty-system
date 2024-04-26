package model

type OrderStatus string

const (
	StatusNewOrder        OrderStatus = `NEW`
	StatusProcessingOrder OrderStatus = `PROCESSING`
	StatusInvalidOrder    OrderStatus = `INVALID`
	StatusProcessedOrder  OrderStatus = `PROCESSED`
)

type AccrualOrderStatus string

const (
	StatusRegisteredAccrual AccrualOrderStatus = `REGISTERED`
	StatusInvalidAccrual    AccrualOrderStatus = `INVALID`
	StatusProcessingAccrual AccrualOrderStatus = `PROCESSING`
	StatusProcessedAccrual  AccrualOrderStatus = `PROCESSED`
)

type UploadedOrders []UploadedOrder

type UploadedOrder struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	Accrual    int    `json:"accrual"`
	UploadTime string `json:"uploaded_at"`
}
