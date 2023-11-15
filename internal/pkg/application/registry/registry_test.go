package registry

import (
	"bytes"
	"context"
	"testing"

	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/matryer/is"
)

func TestXxx(t *testing.T) {
	is := is.New(t)

	input := `functionID;type;cipfunction
	function123;stopwatch;pumpbrunn`

	_, err := NewRegistry(context.Background(), bytes.NewBufferString(input), &database.StorageMock{
		CreateFunc: func(ctx context.Context, id string, value any) error {
			return nil
		},
	})
	is.NoErr(err)

	//matches, err := reg.Find(context.Background())
}
