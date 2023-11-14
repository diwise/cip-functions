package functions

type Function interface {
	ID() string
	Handle()
}

type fnct struct{}

func (f *fnct) Handle() {}
