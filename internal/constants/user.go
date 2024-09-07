package constants

import "errors"

// Http Error Code
const (
	UserSignUpFailedErrorCode = 400001 + iota
	UserExistErrorCode
	UserNotExistErrorCode
)

var (
	ErrorUserExist      = errors.New("user already exists, please check your username or mobile, or try to login")
	ErrorUserNotExist   = errors.New("user not exists")
	ErrorUserSignUpFail = errors.New("user sign up fail")
)

// MySQL Error Code
var (
	// User Model
	ErrCodeDuplicateUserNameOrMobileNumber uint16 = 1062
	ErrCodeDuplicateUserNameOrMobile              = errors.New("duplicate username or mobile")
	ErrUserNotFound                               = errors.New("user not found")
)
