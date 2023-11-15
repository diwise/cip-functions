package functions

import (
	"bufio"
	"context"
	"io"
	"strings"

	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type Registry interface {
	Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]Function, error)
	Get(ctx context.Context, functionID string) (Function, error)
}

func NewRegistry(ctx context.Context, input io.Reader, storage database.Storage) (Registry, error) {
	r := &reg{
		f: make(map[string]Function),
	}

	//var err error

	//numErrors := 0
	numFunctions := 0

	logger := logging.GetFromContext(ctx)

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		tokens := strings.Split(line, ";")
		tokenCount := len(tokens)

		if tokenCount >= 1 {
			f := &fnct{
				ID_:       tokens[0],
				Type_:     tokens[1],
				Function_: tokens[2],
			}

			database.CreateOrUpdate[fnct](ctx, storage, f.ID_, *f)

			r.f[tokens[0]] = f
			numFunctions++
		}

	}

	logger.Info("loaded functions from config file", "count", numFunctions)

	return r, nil
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
