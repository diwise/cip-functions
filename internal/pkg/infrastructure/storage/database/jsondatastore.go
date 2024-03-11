package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/jackc/pgx/v5/pgxpool"
)

type JsonDataStore struct {
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

func Connect(ctx context.Context, cfg Config) (*JsonDataStore, error) {
	conn, err := pgxpool.New(ctx, cfg.ConnStr())
	if err != nil {
		return nil, err
	}

	err = conn.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return &JsonDataStore{
		db: conn,
	}, nil
}

func (jds *JsonDataStore) Close() {
	jds.db.Close()
}

func (jds *JsonDataStore) Initialize(ctx context.Context) error {
	return jds.createTables(ctx)
}

func (jds *JsonDataStore) Create(ctx context.Context, id string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = jds.db.Exec(ctx, `insert into cip_fnct (id, data) values ($1, $2)`, id, string(b))
	if err != nil {
		return err
	}

	return nil
}

func (jds *JsonDataStore) Delete(ctx context.Context, id string) error {
	_, err := jds.db.Exec(ctx, `delete from cip_fnct where id = $1`, id)
	if err != nil {
		return err
	}

	_, err = jds.db.Exec(ctx, `delete from cip_fnct_values where cip_fnct_id = $1`, id)
	if err != nil {
		return err
	}

	return nil
}

func (jds *JsonDataStore) Update(ctx context.Context, id string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	_, err = jds.db.Exec(ctx, `update cip_fnct set data = $2 where id = $1`, id, string(b))
	if err != nil {
		return err
	}

	return nil
}

func (jds *JsonDataStore) Read(ctx context.Context, id string) (any, error) {	
	var obj any
	
	err := jds.db.QueryRow(ctx, `select data from cip_fnct where id = $1`, id).Scan(&obj)		
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (jds *JsonDataStore) Exists(ctx context.Context, id string) bool {
	var n int32
	err := jds.db.QueryRow(ctx, `select count(*) from cip_fnct where id = $1`, id).Scan(&n)
	if err != nil {
		return false
	}

	return n > 0
}

func (jds *JsonDataStore) createTables(ctx context.Context) error {
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

	tx, err := jds.db.Begin(ctx)
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
