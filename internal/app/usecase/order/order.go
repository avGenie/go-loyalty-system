package order

import (
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
)

func IsUpdatableAccrualStatus(newStatus, currentStatus entity.OrderStatus) bool {
	if entity.StatusNewOrder == newStatus {
		return false
	}

	if entity.StatusInvalidOrder == newStatus && entity.StatusInvalidOrder != currentStatus {
		return true
	}

	if entity.StatusProcessingOrder == newStatus && entity.StatusNewOrder == currentStatus {
		return true
	}

	if entity.StatusProcessedOrder == newStatus && (entity.StatusNewOrder == currentStatus ||
		entity.StatusProcessingOrder == currentStatus) {
		return true
	}

	return false
}
