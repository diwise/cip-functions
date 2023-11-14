package database

import (
	"context"
	"fmt"

	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage interface {
	Initialize(context.Context) error
	//Add(ctx context.Context, id, label string, value float64, timestamp time.Time) error
	AddFnct(ctx context.Context, id, fnType, cipfnctType string) error
	//History(ctx context.Context, id, label string, lastN int) ([]LogValue, error)
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

func (i *impl) Initialize(ctx context.Context) error {
	return i.createTables(ctx)
}

func (i *impl) createTables(ctx context.Context) error {
	ddl := `
		CREATE TABLE IF NOT EXISTS fnct (
			id 		  TEXT PRIMARY KEY NOT NULL,
			type 	  TEXT NOT NULL,
			sub_type  TEXT NOT NULL,
			tenant 	  TEXT NOT NULL,
			source 	  TEXT NULL,
			latitude  NUMERIC(7, 5),
			longitude NUMERIC(7, 5),
			cip_function TEXT NOT NULL
	  	);

		CREATE TABLE IF NOT EXISTS fnct_history (
			row_id 	bigserial,
			time 	TIMESTAMPTZ NOT NULL,
			fnct_id TEXT NOT NULL,
			label 	TEXT NOT NULL,
			value 	DOUBLE PRECISION NOT NULL,
			FOREIGN KEY (fnct_id) REFERENCES fnct (id)
	  	);

		CREATE INDEX IF NOT EXISTS fnct_history_fnct_id_label_idx ON fnct_history (fnct_id, label);`

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
		WHERE hypertable_name = 'fnct_history';`).Scan(&n)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	if n == 0 {
		_, err := tx.Exec(ctx, `SELECT create_hypertable('fnct_history', 'time');`)
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

func (i *impl) AddFnct(ctx context.Context, id, fnType, cipfnType string) error {
	_, err := i.db.Exec(ctx, `
		INSERT INTO fnct(id,type,cip_function) VALUES ($1,$2,$3) ON CONFLICT (id) DO NOTHING;
	`, id, fnType, cipfnType)

	return err
}
