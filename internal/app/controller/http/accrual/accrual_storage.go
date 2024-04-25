package accrual

import (
	"container/list"
	"errors"
	"sync"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
)

const (
	storageFreeSpace = 10
)

var (
	ErrAddElement   = errors.New("couldn't add element to accrual storage")
	ErrGetElement   = errors.New("couldn't get element from accrual storage")
	ErrEmptyStorage = errors.New("accrual storage is empty")

	ErrEmptyStorageSpace = errors.New("accrual storage space is empty")
)

type AccrualStorage struct {
	mutex sync.Mutex
	list  *list.List

	freeSpace int
}

func NewStorage() *AccrualStorage {
	return &AccrualStorage{
		mutex:     sync.Mutex{},
		list:      list.New(),
		freeSpace: storageFreeSpace,
	}
}

func (s *AccrualStorage) Add(number entity.OrderNumber) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.freeSpace == 0 {
		return ErrEmptyStorageSpace
	}
	el := s.list.PushBack(number)
	s.freeSpace--

	if el.Value != number {
		return ErrAddElement
	}

	return nil
}

func (s *AccrualStorage) Get() (entity.OrderNumber, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.list.Len() == 0 {
		return entity.OrderNumber(""), ErrEmptyStorage
	}

	el := s.list.Front()
	s.freeSpace++

	number, ok := el.Value.(entity.OrderNumber)
	if !ok {
		return entity.OrderNumber(""), ErrGetElement
	}

	return number, nil
}
