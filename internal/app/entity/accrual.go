package entity

import "time"

type AccrualProcessingResponse struct {
	Order      Order
	RetryAfter time.Duration
}
