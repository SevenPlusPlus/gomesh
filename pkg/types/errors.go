package types

import "github.com/pkg/errors"

var (
	ErrWouldBlock		= errors.New("would block")
	ErrServerConnClosed = errors.New("server conn is closed")
	ErrServerClosed    = errors.New("Server is closed")
)