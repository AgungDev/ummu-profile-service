package http

import (
	nethttp "net/http"

	ummuauth "github.com/AgungDev/ummu-go-auth"
	"github.com/gin-gonic/gin"
)

func JWTMiddleware(authenticator *ummuauth.Auth) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		continued := false
		next := nethttp.HandlerFunc(func(_ nethttp.ResponseWriter, request *nethttp.Request) {
			continued = true
			ctx.Request = request
			ctx.Next()
		})
		authenticator.Middleware(next).ServeHTTP(ctx.Writer, ctx.Request)
		if !continued {
			ctx.Abort()
		}
	}
}
