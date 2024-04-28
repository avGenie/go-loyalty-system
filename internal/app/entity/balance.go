package entity

type UserBalances []UserBalance

type UserBalance struct {
	UserID  UserID
	Balance float64
}
