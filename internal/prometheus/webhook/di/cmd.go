package di

import (
	"github.com/gin-gonic/gin"
)

type Cmd struct {
	Server *gin.Engine
	Start  func()
}
