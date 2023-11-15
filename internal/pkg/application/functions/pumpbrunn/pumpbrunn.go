package pumpbrunn

const FunctionTypeName string = "pumpbrunnar"

type Pumpbrunn interface {
	ID() string
	Handle() error
	State() bool
}

type pb struct {
	id_    string `json:"id"`
	state_ bool   `json:"state"`
}

func New(id string) Pumpbrunn {
	p := &pb{
		id_:    id,
		state_: false,
	}

	return p
}

func (pb *pb) ID() string {
	return pb.id_
}

func (pb *pb) State() bool {
	return pb.state_
}

func (pb *pb) Handle() error {
	return nil
}
