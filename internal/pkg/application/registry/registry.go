package registry

import (
	"bufio"
	"context"
	"io"
	"strings"

	"github.com/diwise/cip-functions/internal/pkg/application/functions"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

//go:generate moq -rm -out registry_mock.go . Registry
type Registry interface {
	Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]functions.Function, error)
}

func NewRegistry(ctx context.Context, input io.Reader, storage database.Storage) (Registry, error) {
	r := &reg{
		f: make(map[string]functions.Function),
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

		//TODO: parse file and add function mappings to registry

		if tokenCount >= 1 {

			numFunctions++
		}

	}

	logger.Info("loaded functions from config file", "count", numFunctions)

	return r, nil
}

type reg struct {
	f map[string]functions.Function
}

func (r *reg) Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]functions.Function, error) {
	fn := make([]functions.Function, 0)
	
	for _, m := range matchers {
		// TODO: check for duplicates before appending
		fn = append(fn, m(r)...)
	}
	
	return fn, nil
}

type RegistryMatcherFunc func(r *reg) []functions.Function

func FindByID(functionId string) RegistryMatcherFunc {
	return func(r *reg) []functions.Function {
		
		// TODO: implement logic to find function by ID
		
		return nil
	}
}
