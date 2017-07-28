package snmp_subagent

import (
	"time"

	"github.com/gin-gonic/gin"
)

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()
		end := time.Now()
		latency := end.Sub(start)

		client_ip := c.ClientIP()
		method := c.Request.Method
		status_code := c.Writer.Status()
		log.Infof("[GIN] %v | %v %v | %v (%v)", client_ip, method, path, status_code, latency)

		//[GIN] 2017/01/30 - 16:24:38 | 204 |    1.460277ms | 172.17.0.1 |   DELETE  /applications/

	}
}
