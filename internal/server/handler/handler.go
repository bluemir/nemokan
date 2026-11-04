package handler

import (
	"github.com/bluemir/nemokan/internal/server/injector"
)

var (
	backends = injector.Backends
	mailer   = injector.Mail
)

type ListResponse[T any] struct {
	Items []T
	//Page int
	//PageSize int
}
