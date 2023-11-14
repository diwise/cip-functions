package pumpbrunn

const FunctionTypeName string = "pumpbrunnar"

type Pumpbrunn interface {
	State() bool
}

type pb struct {
	State_ bool `json:"state"`
}

func New() Pumpbrunn {
	p := &pb{
		State_: false,
	}

	return p
}

func (pb *pb) State() bool {
	return pb.State_
}
