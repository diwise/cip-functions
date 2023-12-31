package functions

import (
	"bytes"
	"context"
	"testing"

	"github.com/matryer/is"
)

func TestNew(t *testing.T) {
	is := is.New(t)

	ctx := context.Background()

	_, err := NewRegistry(ctx, bytes.NewBufferString(cip_functions_csv))
	is.NoErr(err)
}

func TestFind(t *testing.T) {
	is := is.New(t)

	ctx := context.Background()

	reg, err := NewRegistry(ctx, bytes.NewBufferString(cip_functions_csv))
	is.NoErr(err)

	items, err := reg.Find(ctx, FindByFunctionID("fnID:002"))
	is.NoErr(err)

	is.Equal(len(items), 1)
	is.Equal(items[0].FnID, "fnID:002")
	is.Equal(items[0].Type, "combinedsewageoverflow")
	is.Equal(len(items[0].Options), 3)
}

func TestFindSewagePumpingStation(t *testing.T) {
	is := is.New(t)

	ctx := context.Background()

	reg, err := NewRegistry(ctx, bytes.NewBufferString(cip_functions_csv))
	is.NoErr(err)

	items, err := reg.Find(ctx, FindByFunctionID("stopwatch:002"))
	is.NoErr(err)

	is.Equal(len(items), 1)
	is.Equal(items[0].FnID, "stopwatch:002")
	is.Equal(items[0].Type, "sewagepumpingstation")
}

const cip_functions_csv string = `iot-functionID;cip-function_type;arguments
fnID:001;combinedsewageoverflow;
fnID:002;combinedsewageoverflow;cipID=abc,max=100,min=0
stopwatch:002;sewagepumpingstation;`
