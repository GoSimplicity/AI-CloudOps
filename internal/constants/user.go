package constants

import "errors"

// Http Error Code
const (
	UserSignUpFailedErrorCode = 400001 + iota
	UserExistErrorCode
	UserNotExistErrorCode
)

var (
	ErrorUserExist         = errors.New("user already exists, check your username or mobile, or try to login")
	ErrorUserNotExist      = errors.New("user not exists")
	ErrorUserSignUpFail    = errors.New("user sign up fail")
	ErrorPasswordIncorrect = errors.New("user password incorrect")
)
