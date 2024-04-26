package entity

type Orders []Order

type OrderNumber string

type Order struct {
	Number      OrderNumber
	Status      string
	Accrual     float64
	DateCreated string
}
