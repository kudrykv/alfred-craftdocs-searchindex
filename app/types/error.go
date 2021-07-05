package types

type Error struct {
	Title string
	Err   error
}

func (e Error) Error() string {
	return e.Err.Error()
}

func NewError(title string, err error) Error {
	return Error{Title: title, Err: err}
}
