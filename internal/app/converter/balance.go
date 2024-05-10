package converter

import (
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/model"
)

func ConvertStorageBalanceToOutput(balance entity.UserBalance) model.UserBalanceResponse {
	return model.UserBalanceResponse{
		Current: balance.Balance,
		Withdrawn: balance.Withdrawans,
	}
}