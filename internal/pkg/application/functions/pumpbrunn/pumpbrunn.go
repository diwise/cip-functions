package pumpbrunn

const FunctionTypeName string = "pumpbrunnar"

type Pumpbrunn interface {
	ID() string
	Handle() error
	State() bool
}

type pb struct {
	ID_    string `json:"id"`
	State_ bool   `json:"state"`
}

func New(id string) Pumpbrunn {
	p := &pb{
		ID_:    id,
		State_: false,
	}

	return p
}

func (pb *pb) ID() string {
	return pb.ID_
}

func (pb *pb) State() bool {
	return pb.State_
}

func (pb *pb) Handle() error {
	return nil
}
