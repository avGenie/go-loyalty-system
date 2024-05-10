package entity

type OrderStatus string

const (
	StatusNewOrder        OrderStatus = `NEW`
	StatusProcessingOrder OrderStatus = `PROCESSING`
	StatusInvalidOrder    OrderStatus = `INVALID`
	StatusProcessedOrder  OrderStatus = `PROCESSED`
)

type UpdateUserOrders []UpdateUserOrder

type UpdateUserOrder struct {
	UserID UserID
	Order  Order
}

type Orders []Order

type OrderNumber string

type Order struct {
	Number      OrderNumber
	Status      OrderStatus
	Accrual     float64
	DateCreated string
}
