package accrual

import "github.com/avGenie/go-loyalty-system/internal/app/entity"

type AccrualConnector struct {
	inputChan      chan string
	outputChan     chan entity.Order
}

func NewConnector() *AccrualConnector {
	return &AccrualConnector{
		inputChan: make(chan string),
		outputChan: make(chan entity.Order),
	}
}

func (c *AccrualConnector) SetInput(number string) {
	c.inputChan <- number
}

func (c *AccrualConnector) GetInput() (string, bool) {
	val, ok := <- c.inputChan

	return val, ok
}

func (c *AccrualConnector) CloseInput() {
	close(c.inputChan)
}

func (c *AccrualConnector) SetOutput(order entity.Order) {
	c.outputChan <- order
}

func (c *AccrualConnector) GetOutput() (entity.Order, bool) {
	val, ok := <- c.outputChan

	return val, ok
}

func (c *AccrualConnector) CloseOutput() {
	close(c.outputChan)
}
