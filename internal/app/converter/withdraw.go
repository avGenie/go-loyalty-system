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

func ConvertWithdrawToWithdrawResponse(withdrawals entity.Withdrawals) model.WithdrawalsResponses {
	responses := make(model.WithdrawalsResponses, 0, len(withdrawals))
	for _, withdraw := range withdrawals {
		response := model.WithdrawalsResponse{
			OrderNumber: string(withdraw.OrderNumber),
			Sum: withdraw.Sum,
			DateCreated: withdraw.DateCreated,
		}

		responses = append(responses, response)
	}

	return responses
}