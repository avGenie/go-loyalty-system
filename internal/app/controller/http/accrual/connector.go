package accrual

import "github.com/avGenie/go-loyalty-system/internal/app/entity"

type AccrualConnector struct {
	inputChan      chan entity.AccrualOrderRequest
	outputChan     chan entity.AccrualOrder
}

func NewConnector() *AccrualConnector {
	return &AccrualConnector{
		inputChan: make(chan entity.AccrualOrderRequest),
		outputChan: make(chan entity.AccrualOrder),
	}
}

func (c *AccrualConnector) SetInput(number entity.AccrualOrderRequest) {
	c.inputChan <- number
}

func (c *AccrualConnector) GetInput() (entity.AccrualOrderRequest, bool) {
	val, ok := <- c.inputChan

	return val, ok
}

func (c *AccrualConnector) CloseInput() {
	close(c.inputChan)
}

func (c *AccrualConnector) SetOutput(order entity.AccrualOrder) {
	c.outputChan <- order
}

func (c *AccrualConnector) GetOutput() (entity.AccrualOrder, bool) {
	val, ok := <- c.outputChan

	return val, ok
}

func (c *AccrualConnector) CloseOutput() {
	close(c.outputChan)
}
