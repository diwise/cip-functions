package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate moq -rm -out database_mock.go . Storage
type Storage interface {
	Initialize(ctx context.Context) error
	Close()
	Select(ctx context.Context, id string) (any, error)
	Create(ctx context.Context, id string, value any) error
	Update(ctx context.Context, id string, value any) error
	Remove(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) bool
}

type impl struct {
	db *pgxpool.Pool
}

type Config struct {
	host     string
	user     string
	password string
	port     string
	dbname   string
	sslmode  string
}

func (c Config) ConnStr() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", c.user, c.password, c.host, c.port, c.dbname, c.sslmode)
}

func LoadConfiguration(ctx context.Context) Config {
	return Config{
		host:     env.GetVariableOrDefault(ctx, "POSTGRES_HOST", ""),
		user:     env.GetVariableOrDefault(ctx, "POSTGRES_USER", ""),
		password: env.GetVariableOrDefault(ctx, "POSTGRES_PASSWORD", ""),
		port:     env.GetVariableOrDefault(ctx, "POSTGRES_PORT", "5432"),
		dbname:   env.GetVariableOrDefault(ctx, "POSTGRES_DBNAME", "diwise"),
		sslmode:  env.GetVariableOrDefault(ctx, "POSTGRES_SSLMODE", "disable"),
	}
}

func Connect(ctx context.Context, cfg Config) (Storage, error) {
	conn, err := pgxpool.New(ctx, cfg.ConnStr())
	if err != nil {
		return nil, err
	}

	err = conn.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return &impl{
		db: conn,
	}, nil
}

func (i *impl) Close() {
	i.db.Close()
}

func (i *impl) Initialize(ctx context.Context) error {
	return i.createTables(ctx)
}

func (i *impl) Create(ctx context.Context, id string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = i.db.Exec(ctx, `insert into cip_fnct (id, data) values ($1, $2)`, id, string(b))
	if err != nil {
		return err
	}

	return nil
}

func (i *impl) Remove(ctx context.Context, id string) error {
	_, err := i.db.Exec(ctx, `delete from cip_fnct where id = $1`, id)
	if err != nil {
		return err
	}

	_, err = i.db.Exec(ctx, `delete from cip_fnct_values where cip_fnct_id = $1`, id)
	if err != nil {
		return err
	}

	return nil
}

func (i *impl) Update(ctx context.Context, id string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = i.db.Exec(ctx, `update cip_fnct set data = $2 where id = $1`, id, string(b))
	if err != nil {
		return err
	}

	return nil
}

func (i *impl) Select(ctx context.Context, id string) (any, error) {
	row := i.db.QueryRow(ctx, `select data from cip_fnct where id = $1`, id)

	var obj any
	err := row.Scan(&obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func Get[T any](ctx context.Context, storage Storage, id string) (T, error) {
	t1, err := storage.Select(ctx, id)
	if err != nil {
		return *new(T), err
	}

	b, err := json.Marshal(t1)
	if err != nil {
		return *new(T), err
	}

	t := *new(T)

	err = json.Unmarshal(b, &t)
	if err != nil {
		return *new(T), err
	}

	return t, nil
}

func GetOrDefault[T any](ctx context.Context, storage Storage, id string, defaultValue T) (T, error) {
	t, err := Get[T](ctx, storage, id)
	if err != nil {
		return defaultValue, nil
	}

	return t, nil
}

func CreateOrUpdate[T any](ctx context.Context, storage Storage, id string, value T) error {
	var err error

	if storage.Exists(ctx, id) {
		err = storage.Update(ctx, id, value)
	} else {
		err = storage.Create(ctx, id, value)
	}

	if err != nil {
		return err
	}

	return nil
}

func (i *impl) Exists(ctx context.Context, id string) bool {
	var n int32
	err := i.db.QueryRow(ctx, `select count(*) from cip_fnct where id = $1`, id).Scan(&n)
	if err != nil {
		return false
	}

	return n > 0
}

func (i *impl) createTables(ctx context.Context) error {
	ddl := `
		CREATE TABLE IF NOT EXISTS cip_fnct (
			id 		  TEXT PRIMARY KEY NOT NULL,
			data      JSONB NOT NULL
	  	);
		
		CREATE TABLE IF NOT EXISTS cip_fnct_values (
			time 			TIMESTAMPTZ NOT NULL,
			cip_fnct_id 	TEXT NOT NULL,
			name			TEXT NOT NULL,
			value			NUMERIC NULL,
			value_string 	TEXT NULL,
			value_bool 		BOOLEAN NULL
		);`

	tx, err := i.db.Begin(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, ddl)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	var n int32
	err = tx.QueryRow(ctx, `
		SELECT COUNT(*) n
		FROM timescaledb_information.hypertables
		WHERE hypertable_name = 'cip_fnct_values';`).Scan(&n)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	if n == 0 {
		_, err := tx.Exec(ctx, `SELECT create_hypertable('cip_fnct_values', 'time');`)
		if err != nil {
			tx.Rollback(ctx)
			return err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}
