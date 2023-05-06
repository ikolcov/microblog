package models

import "errors"

var ErrBadRequest = errors.New("bad page token")
var ErrUnauthorized = errors.New("user token is invalid")
var ErrFobidden = errors.New("user is not allowed to edit this post")
var ErrNotFound = errors.New("post is not found")
