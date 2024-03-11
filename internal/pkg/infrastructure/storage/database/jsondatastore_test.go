package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestInitialize(t *testing.T) {
	is, s, _, connected, err := testSetup(t)
	if !connected {
		t.Skip("not connected")
	}
	is.NoErr(err)
	s.Close()
}

func TestCreate(t *testing.T) {
	is, s, ctx, connected, err := testSetup(t)
	if !connected {
		t.Skip("not connected")
	}
	is.NoErr(err)
	defer s.Close()

	err = s.Create(ctx, "id:001", struct {
		Name string
		Age  int
	}{"John", 30})

	is.NoErr(err)
}

func TestSelect(t *testing.T) {
	is, s, ctx, connected, err := testSetup(t)
	if !connected {
		t.Skip("not connected")
	}
	is.NoErr(err)
	defer s.Close()

	id := fmt.Sprintf("id:%d", time.Now().UnixNano())

	err = s.Create(ctx, id, struct {
		Name string
		Age  int
	}{"John", 30})
	is.NoErr(err)

	v, err := s.Read(ctx, id)
	is.NoErr(err)

	is.True(v != nil)
}

func testSetup(t *testing.T) (*is.I, *JsonDataStore, context.Context, bool, error) {
	is := is.New(t)
	ctx := context.Background()
	s, err := Connect(ctx, Config{
		host:     "localhost",
		user:     "postgres",
		password: "postgres",
		port:     "5432",
		dbname:   "postgres",
		sslmode:  "disable",
	})
	if err != nil {
		return nil, nil, nil, false, err
	}

	err = s.Initialize(ctx)
	return is, s, ctx, true, err
}
