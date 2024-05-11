package entity

import (
	"time"
)

type AccrualStatus int

const (
	StatusProcessing AccrualStatus = iota
	StatusPause
	StatusOrderNotRegistered
	StatusOK
	StatusError
)

type AccrualProcessingResponse struct {
	Order      Order
	RetryAfter time.Duration
}

type AccrualOrderRequest struct {
	Number OrderNumber
	UserID UserID
}

type AccrualOrder struct {
	Order      Order
	UserID     UserID
	RetryAfter time.Duration
	Status     AccrualStatus
}

func CreateAccrualRequest(userID UserID, number OrderNumber) AccrualOrderRequest {
	return AccrualOrderRequest{
		Number: number,
		UserID: userID,
	}
}

func CreateProcessingAccrualOrder(userID UserID, order Order) AccrualOrder {
	return AccrualOrder{
		Order:  order,
		UserID: userID,
		Status: StatusProcessing,
	}
}

func CreatePausedAccrualOrder() AccrualOrder {
	return AccrualOrder{
		Order:  Order{},
		Status: StatusPause,
	}
}
