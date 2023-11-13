package functions

type Function interface {
	Handle()
}

type fnct struct{}

func (f *fnct) Handle() {}
