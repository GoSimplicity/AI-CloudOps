package di

import (
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

type Cmd struct {
	Server *gin.Engine
	Cron   *cron.Cron
}
