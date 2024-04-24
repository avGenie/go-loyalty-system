package entity

import "time"

type AccrualStatus int

const (
	StatusProcessing AccrualStatus = iota
	StatusPause
)

type AccrualProcessingResponse struct {
	Order      Order
	RetryAfter time.Duration
}

type AccrualOrder struct {
	Order  Order
	Status AccrualStatus
}

func CreateProcessingAccrualOrder(order Order) AccrualOrder {
	return AccrualOrder{
		Order:  order,
		Status: StatusProcessing,
	}
}

func CreatePausedAccrualOrder() AccrualOrder {
	return AccrualOrder{
		Order:  Order{},
		Status: StatusPause,
	}
}
