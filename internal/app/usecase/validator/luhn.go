package validator

import (
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/joeljunstrom/go-luhn"
)

func OrderNumberValidation(number entity.OrderNumber) bool {
	return luhn.Valid(string(number))
}
