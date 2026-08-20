package apperr

type Error struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *Error) Error() string { return e.Message }
