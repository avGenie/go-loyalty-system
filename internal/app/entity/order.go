package entity

import "time"

type Orders []Order

type OrderNumber string

type Order struct {
	Number      OrderNumber
	Status      int
	Accrual     float64
	DateCreated time.Time
}
