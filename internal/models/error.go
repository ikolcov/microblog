package models

import "errors"

var ErrUnauthorized = errors.New("user token is invalid")
var ErrNotFound = errors.New("post is not found")
var ErrBadRequest = errors.New("bad page token")
