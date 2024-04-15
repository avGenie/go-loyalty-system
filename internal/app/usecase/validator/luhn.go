package validator

import (
	"github.com/ShiraazMoollatjie/goluhn"
	"github.com/avGenie/go-loyalty-system/internal/app/entity"
)

func OrderNumberValidation(number entity.OrderNumber) error {
	return goluhn.Validate(string(number))
}