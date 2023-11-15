package functions

import (
	"context"
	"io"

	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
)

type Registry interface {
	Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]Function, error)
	Get(ctx context.Context, functionID string) (Function, error)
}

func NewRegistry(ctx context.Context, input io.Reader, storage database.Storage) (Registry, error) {
	return &reg{
		f: make(map[string]Function),
	}, nil
}

type reg struct {
	f map[string]Function
}

func (r *reg) Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]Function, error) {
	return nil, nil
}

func (r *reg) Get(ctx context.Context, functionID string) (Function, error) {
	return nil, nil
}

type RegistryMatcherFunc func(r *reg) []Function

func MatchFunction(functionId string) RegistryMatcherFunc {
	return nil
}
