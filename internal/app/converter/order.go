package converter

import (
	"fmt"
	"time"

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
			Status:     order.Status,
			Accrual:    order.Accrual,
			UploadTime: timeCreated.String(),
		}
		uploadedOrders = append(uploadedOrders, uploadedOrder)
	}

	return uploadedOrders, nil
}
