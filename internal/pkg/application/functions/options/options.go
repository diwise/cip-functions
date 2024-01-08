package options

type Option struct {
	Key string
	Val string
}

func Exists(options []Option, key string) (string, bool) {
	for _, o := range options {
		if o.Key == key {
			return o.Val, true
		}
	}

	return "", false
}
