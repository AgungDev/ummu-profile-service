package http

import (
	nethttp "net/http"

	"github.com/gin-gonic/gin"
)

func LimitRequestBody(limit int64) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Request.Body = nethttp.MaxBytesReader(ctx.Writer, ctx.Request.Body, limit)
		ctx.Next()
	}
}
