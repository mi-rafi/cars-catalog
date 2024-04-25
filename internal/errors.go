package internal

import "fmt"

type ClientError struct {
	Code int
	Msg  string
	Err  error
}

func (c ClientError) Error() string {
	return fmt.Sprintf("error from client: %s, code: %d, err: %v", c.Msg, c.Code, c.Err)
}
