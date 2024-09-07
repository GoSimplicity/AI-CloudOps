package constants

import "errors"

const (
	UserSignFailedErrorCode = 400001 + iota
	UserExistErrorCode
	UserNotExistErrorCode
)

var (
	ErrorUserExist    = errors.New("user already exists")
	ErrorUserNotExist = errors.New("user not exists")
	ErrorUserSignFail = errors.New("user sign up fail")
)
