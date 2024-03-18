package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

//go:generate moq -rm -out storage_mock.go . Storage
type Storage interface {
	Create(ctx context.Context, id, typeName string, value any) error
	Read(ctx context.Context, id, typeName string) (any, error)
	Update(ctx context.Context, id, typeName string, value any) error
	Delete(ctx context.Context, id, typeName string) error
	Exists(ctx context.Context, id, typeName string) bool
}

func Get[T any](ctx context.Context, storage Storage, id string) (T, error) {
	typeName := GetTypeName[T]()

	t1, err := storage.Read(ctx, id, typeName)
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

	typeName := GetTypeName[T]()

	if storage.Exists(ctx, id, typeName) {
		err = storage.Update(ctx, id, typeName, value)
	} else {
		err = storage.Create(ctx, id, typeName, value)
	}

	if err != nil {
		return err
	}

	return nil
}

func GetTypeName[T any]() string {
	t := *new(T)
	typeName := fmt.Sprintf("%T", t)
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		typeName = parts[len(parts)-1]
	}
	return typeName
}
