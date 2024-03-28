package things

import (
	"encoding/json"
	"testing"

	"github.com/matryer/is"
)


type thingWithProps struct {
	Thing
	Prop1 string
	Prop2 float64	
}


func TestUnmarshalToThing(t *testing.T){
	is := is.New(t)
	
	data, _ := json.Marshal(thingWithProps{
		Thing: Thing{
			Id: "id",
			Type: "type",
			Location: Location{
				Latitude: 62,
				Longitude: 17,
			},
			Tenant: "default",
		},
		Prop1: "prop1",
		Prop2: 2,
	})


	t_ := Thing{}

	err := json.Unmarshal(data, &t_)
	is.NoErr(err)

	is.True(t_.Properties != nil)
}