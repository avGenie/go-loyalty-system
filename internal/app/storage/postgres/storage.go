package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/storage/api/model"
)

type Postgres struct {
	model.Storage

	db *sql.DB
}

func NewPostgresStorage(dbStorageConnect string) (*Postgres, error) {
	db, err := sql.Open("pgx", dbStorageConnect)
	if err != nil {
		return nil, fmt.Errorf("error while postgresql connect: %w", err)
	}

	return &Postgres{
		db: db,
	}, nil
}

func (s *Postgres) Ping(ctx context.Context) error {
	return nil
}

func (s *Postgres) CreateUser(user entity.User) error {
	return nil
}

func (s *Postgres) GetUser(user entity.User) (entity.User, error) {
	return entity.User{}, nil
}
