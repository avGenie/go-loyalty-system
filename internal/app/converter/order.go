package converter

import (
	"fmt"
	"time"

	"github.com/golang-module/carbon/v2"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/model"
)

func ConvertStorageOrdersToOutputUploadedOrders(orders entity.Orders) (model.UploadedOrders, error) {
	uploadedOrders := make(model.UploadedOrders, 0, len(orders))

	for _, order := range orders {
		timeCreated, err := time.Parse(time.RFC3339, order.DateCreated)
		if err != nil {
			return nil, fmt.Errorf("error while transform time to RFC3339 format")
		}

		uploadedOrder := model.UploadedOrder{
			Number:     string(order.Number),
			Status:     string(order.Status),
			Accrual:    order.Accrual,
			UploadTime: carbon.Parse(timeCreated.String()).ToRfc3339String(),
		}
		uploadedOrders = append(uploadedOrders, uploadedOrder)
	}

	return uploadedOrders, nil
}

func ConvertAccrualResponseToOrder(response model.AccrualResponse) entity.Order {
	return entity.Order{
		Number: entity.OrderNumber(response.Number),
		Status: ConvertAccrualStatusToAPI(model.AccrualOrderStatus(response.Status)),
		Accrual: response.Accrual,
	}
}

func ConvertAccrualStatusToAPI(accrualStatus model.AccrualOrderStatus) entity.OrderStatus {
	switch accrualStatus {
	case model.StatusInvalidAccrual:
		return entity.StatusInvalidOrder
	case model.StatusRegisteredAccrual:
		return entity.StatusNewOrder
	case model.StatusProcessingAccrual:
		return entity.StatusProcessingOrder
	case model.StatusProcessedAccrual:
		return entity.StatusProcessedOrder
	default:
		return entity.StatusInvalidOrder
	}
}
