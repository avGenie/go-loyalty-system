package entity

type UpdateUserBalances []UserBalance

type UpdateUserBalance struct {
	UserID      UserID
	Balance     float64
}

type UserBalance struct {
	UserID      UserID
	Balance     float64
	Withdrawans float64
}
