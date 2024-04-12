package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	"github.com/avGenie/go-loyalty-system/internal/app/storage/api/model"
	"github.com/pressly/goose/v3"
)

const (
	migrationDB     = "postgres"
	migrationFolder = "migrations"
)

//go:embed migrations/*.sql
var migrationFs embed.FS

type Postgres struct {
	model.Storage

	db *sql.DB
}

func NewPostgresStorage(dbStorageConnect string) (*Postgres, error) {
	db, err := sql.Open("pgx", dbStorageConnect)
	if err != nil {
		return nil, fmt.Errorf("error while postgresql connect: %w", err)
	}

	err = migration(db)
	if err != nil {
		return nil, fmt.Errorf("error while postgresql migration: %w", err)
	}

	return &Postgres{
		db: db,
	}, nil
}

func (s *Postgres) CreateUser(ctx context.Context, user entity.User) error {
	return nil
}

func (s *Postgres) GetUser(ctx context.Context, user entity.User) (entity.User, error) {
	return entity.User{}, nil
}

func migration(db *sql.DB) error {
	goose.SetBaseFS(migrationFs)

	if err := goose.SetDialect(migrationDB); err != nil {
		return err
	}

	if err := goose.Up(db, migrationFolder); err != nil {
		return err
	}

	return nil
}
