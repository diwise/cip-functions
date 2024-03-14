package storage

import (
	"context"
	"encoding/json"
)

//go:generate moq -rm -out storage_mock.go . Storage
type Storage interface {
	Create(ctx context.Context, id string, value any) error
	Read(ctx context.Context, id string) (any, error)
	Update(ctx context.Context, id string, value any) error
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) bool
}

func Get[T any](ctx context.Context, storage Storage, id string) (T, error) {
	t1, err := storage.Read(ctx, id)
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
