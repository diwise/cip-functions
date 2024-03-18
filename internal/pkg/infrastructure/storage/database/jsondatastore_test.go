package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/diwise/cip-functions/internal/pkg/infrastructure/storage"
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

func TestCreateStruct(t *testing.T) {
	is, s, ctx, connected, err := testSetup(t)
	if !connected {
		t.Skip("not connected")
	}
	is.NoErr(err)
	defer s.Close()

	err = s.Create(ctx, fmt.Sprintf("id:%d", time.Now().UnixNano()), struct {
		Name string
		Age  int
	}{"John", 30})

	is.True(err != nil)
}

type person struct {
	Age  int    `json:"age"`
	Name string `json:"name"`
}

func TestGetOrDefault(t *testing.T) {
	is, s, ctx, connected, err := testSetup(t)
	if !connected {
		t.Skip("not connected")
	}
	is.NoErr(err)
	defer s.Close()

	id := fmt.Sprintf("id:%d", time.Now().UnixNano())

	p1, err := storage.GetOrDefault(ctx, s, id, person{Age: 31, Name: "Joe"})
	is.NoErr(err)

	err = s.Create(ctx, id, p1)
	is.NoErr(err)

	p2, err := storage.Get[person](ctx, s, id)
	is.NoErr(err)

	is.Equal(p1.Age, p2.Age)
	is.Equal(p1.Name, p2.Name)
}

func TestCreate(t *testing.T) {
	is, s, ctx, connected, err := testSetup(t)
	if !connected {
		t.Skip("not connected")
	}
	is.NoErr(err)
	defer s.Close()

	err = s.Create(ctx, fmt.Sprintf("id:%d", time.Now().UnixNano()), person{
		Age:  30,
		Name: "John",
	})

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

	err = s.Create(ctx, id, person{
		Age:  30,
		Name: "John",
	})
	is.NoErr(err)

	v, err := s.Read(ctx, id, "person")
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
