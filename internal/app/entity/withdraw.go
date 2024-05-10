package entity

type Withdrawals []Withdraw

type Withdraw struct {
	OrderNumber OrderNumber
	Sum         float64
	DateCreated string
}
