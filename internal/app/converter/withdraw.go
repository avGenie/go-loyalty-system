package converter

import (
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/model"
)

func ConvertRequestWithdrawToEntity(request model.WithdrawRequest) entity.Withdraw {
	return entity.Withdraw{
		OrderNumber: entity.OrderNumber(request.Order),
		Sum: request.Sum,
	}
}