package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

func NewConfig(host, user, password, port, dbname, sslmode string) Config {
	return Config{
		host:     host,
		user:     user,
		password: password,
		port:     port,
		dbname:   dbname,
		sslmode:  sslmode,
	}
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

func (jds *JsonDataStore) Create(ctx context.Context, id, typeName string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	id = strings.ToLower(id)
	typeName = strings.ToLower(typeName)
	cipID := fmt.Sprintf("urn:diwise:%s:%s", strings.ToLower(typeName), strings.ToLower(id))

	_, err = jds.db.Exec(ctx, `insert into cip_fnct (cip_id, id, type, data) values ($1, $2, $3, $4)`, cipID, id, typeName, string(b))
	if err != nil {
		return err
	}

	return nil
}

func (jds *JsonDataStore) Update(ctx context.Context, id, typeName string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	id = strings.ToLower(id)
	typeName = strings.ToLower(typeName)

	_, err = jds.db.Exec(ctx, `update cip_fnct set data = $3, updated_on=CURRENT_TIMESTAMP where id = $1 and type = $2`, id, typeName, string(b))
	if err != nil {
		return err
	}

	return nil
}

func (jds *JsonDataStore) Read(ctx context.Context, id, typeName string) (any, error) {
	var obj any

	id = strings.ToLower(id)
	typeName = strings.ToLower(typeName)

	err := jds.db.QueryRow(ctx, `select data from cip_fnct where id=$1 and type=$2`, id, typeName).Scan(&obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (jds *JsonDataStore) Exists(ctx context.Context, id, typeName string) bool {
	var n int32

	id = strings.ToLower(id)
	typeName = strings.ToLower(typeName)

	err := jds.db.QueryRow(ctx, `select count(*) from cip_fnct where id = $1 and type=$2`, id, typeName).Scan(&n)
	if err != nil {
		return false
	}

	return n > 0
}

func (jds *JsonDataStore) createTables(ctx context.Context) error {
	ddl := `
		CREATE TABLE IF NOT EXISTS cip_fnct (
			cip_id  TEXT NOT NULL,
			id 		TEXT NOT NULL,
			type    TEXT NOT NULL,
			data    JSONB NOT NULL,
			PRIMARY KEY(id, type)
	  	);
		
		ALTER TABLE cip_fnct ADD COLUMN IF NOT EXISTS cip_id TEXT NOT NULL DEFAULT ''; 
		ALTER TABLE cip_fnct ADD COLUMN IF NOT EXISTS created_on timestamp with time zone NULL DEFAULT CURRENT_TIMESTAMP;
		ALTER TABLE cip_fnct ADD COLUMN IF NOT EXISTS updated_on timestamp with time zone NULL;
		`

	tx, err := jds.db.Begin(ctx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, ddl)
	if err != nil {
		tx.Rollback(ctx)
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}
