package api

type ErrSubRequest struct {
	msg string
}

func (e *ErrSubRequest) Error() string {
	return e.msg
}
