package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/avGenie/go-loyalty-system/internal/app/entity"
	err_api "github.com/avGenie/go-loyalty-system/internal/app/storage/api/errors"
	"github.com/avGenie/go-loyalty-system/internal/app/storage/api/model"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
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

func (s *Postgres) Close() error {
	err := s.db.Close()
	if err != nil {
		zap.L().Error("error while closing postgres storage", zap.Error(err))

		return fmt.Errorf("couldn'r closed postgres db: %w", err)
	}

	return nil
}

func (s *Postgres) CreateUser(ctx context.Context, user entity.User) error {
	query := `INSERT INTO users VALUES(@userID, @login, @password)`
	args := pgx.NamedArgs{
		"userID":   user.ID.String(),
		"login":    user.Login,
		"password": user.Password,
	}

	_, err := s.db.ExecContext(ctx, query, args)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
			return fmt.Errorf("error while save url to postgres: %w", err_api.ErrLoginExists)
		}

		return fmt.Errorf("unable to insert row to postgres: %w", err)
	}

	return nil
}

func (s *Postgres) GetUser(ctx context.Context, user entity.User) (entity.User, error) {
	query := `SELECT id, password FROM users WHERE login=@login`
	args := pgx.NamedArgs{
		"login": user.Login,
	}

	row := s.db.QueryRowContext(ctx, query, args)
	if row == nil {
		return user, fmt.Errorf("error while postgres request preparation while getting user")
	}

	if row.Err() != nil {
		return user, fmt.Errorf("error while postgres request execution while getting user: %w", row.Err())
	}

	err := row.Scan(&user.ID, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return user, err_api.ErrLoginNotFound
		}
		return user, fmt.Errorf("error while processing response row in postgres while getting user: %w", err)
	}

	return user, nil
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
