package sessions

import (
	"errors"
)

// Sessions errors
var (
	ErrUnexpectedToken = errors.New("given token violates the storage token format")
	ErrNotFound        = errors.New("no such token registered or already deleted")
	ErrExpired         = errors.New("token expired")
	ErrNoStorage       = errors.New("doesn't have any session storage")
	ErrDataNotValid    = errors.New("providing data not valid")
)
