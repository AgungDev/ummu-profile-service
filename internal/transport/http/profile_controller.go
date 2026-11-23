package http

import (
	"errors"
	"io"
	"log/slog"
	nethttp "net/http"
	"strings"

	ummuauth "github.com/AgungDev/ummu-go-auth"
	"github.com/AgungDev/ummu-profile-service/internal/config"
	"github.com/AgungDev/ummu-profile-service/internal/domain"
	"github.com/AgungDev/ummu-profile-service/internal/domain/dto"
	"github.com/AgungDev/ummu-profile-service/internal/usecases"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProfileController struct {
	usecase usecases.ProfileUsecase
	auth    *ummuauth.Auth
	avatar  config.AvatarConfig
	logger  *slog.Logger
}

func NewProfileController(
	usecase usecases.ProfileUsecase,
	auth *ummuauth.Auth,
	avatar config.AvatarConfig,
	logger *slog.Logger,
) *ProfileController {
	return &ProfileController{usecase: usecase, auth: auth, avatar: avatar, logger: logger}
}

func (c *ProfileController) Register(router *gin.Engine) {
	router.GET("/healthz", c.Health)
	protected := router.Group("/api/v1/profiles")
	protected.Use(JWTMiddleware(c.auth))
	protected.GET("/me", c.GetMe)
	protected.PATCH("/me", LimitRequestBody(1<<20), c.UpdateMe)
	protected.POST("/me/avatar", c.UploadAvatar)
	protected.DELETE("/me/avatar", c.DeleteAvatar)
	protected.GET("/:user_id", c.GetByUserID)
	protected.PATCH("/:user_id", LimitRequestBody(1<<20), c.UpdateByUserID)
}

func (c *ProfileController) Health(ctx *gin.Context) {
	ctx.JSON(nethttp.StatusOK, gin.H{"success": true, "data": gin.H{"status": "ok"}})
}

func (c *ProfileController) GetMe(ctx *gin.Context) {
	userID, ok := subjectID(ctx)
	if !ok {
		return
	}
	profile, err := c.usecase.Get(ctx.Request.Context(), userID)
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, gin.H{"success": true, "data": profile})
}

func (c *ProfileController) UpdateMe(ctx *gin.Context) {
	userID, ok := subjectID(ctx)
	if !ok {
		return
	}
	var req dto.UpdateProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		var maxBytesError *nethttp.MaxBytesError
		if errors.As(err, &maxBytesError) {
			c.writeError(ctx, domain.ErrAvatarTooLarge)
			return
		}
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"success": false, "message": "invalid profile request"})
		return
	}
	profile, err := c.usecase.Update(ctx.Request.Context(), userID, req)
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, gin.H{"success": true, "message": "profile updated", "data": profile})
}

func (c *ProfileController) GetByUserID(ctx *gin.Context) {
	if !hasPermission(ctx, "profile.admin:read") {
		c.forbidden(ctx)
		return
	}
	userID, ok := pathUserID(ctx)
	if !ok {
		return
	}
	profile, err := c.usecase.GetPublic(ctx.Request.Context(), userID)
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, gin.H{"success": true, "data": profile})
}

func (c *ProfileController) UpdateByUserID(ctx *gin.Context) {
	if !hasPermission(ctx, "profile.admin:update") {
		c.forbidden(ctx)
		return
	}
	userID, ok := pathUserID(ctx)
	if !ok {
		return
	}
	var req struct {
		FullName    *string `json:"full_name"`
		PhoneNumber *string `json:"phone_number"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		var maxBytesError *nethttp.MaxBytesError
		if errors.As(err, &maxBytesError) {
			c.writeError(ctx, domain.ErrAvatarTooLarge)
			return
		}
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"success": false, "message": "invalid profile request"})
		return
	}
	_, err := c.usecase.Update(ctx.Request.Context(), userID, dto.UpdateProfileRequest{
		FullName: req.FullName, PhoneNumber: req.PhoneNumber,
	})
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	profile, err := c.usecase.GetPublic(ctx.Request.Context(), userID)
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, gin.H{"success": true, "message": "profile updated", "data": profile})
}

func (c *ProfileController) UploadAvatar(ctx *gin.Context) {
	userID, ok := subjectID(ctx)
	if !ok {
		return
	}
	if c.avatar.Endpoint == "" && c.avatar.LocalDir == "" {
		c.writeError(ctx, domain.ErrAvatarUnavailable)
		return
	}
	requestLimit := c.avatar.MaxBytes + (1 << 20)
	if c.avatar.MaxBytes > 0 && ctx.Request.ContentLength > requestLimit {
		c.writeError(ctx, domain.ErrAvatarTooLarge)
		return
	}
	ctx.Request.Body = nethttp.MaxBytesReader(ctx.Writer, ctx.Request.Body, requestLimit)
	if err := ctx.Request.ParseMultipartForm(c.avatar.MaxBytes); err != nil {
		var maxBytesError *nethttp.MaxBytesError
		if errors.As(err, &maxBytesError) {
			c.writeError(ctx, domain.ErrAvatarTooLarge)
			return
		}
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"success": false, "message": "invalid multipart upload"})
		return
	}
	if ctx.Request.MultipartForm != nil {
		defer ctx.Request.MultipartForm.RemoveAll()
	}
	file, header, err := ctx.Request.FormFile("avatar")
	if err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"success": false, "message": "avatar file is required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, c.avatar.MaxBytes+1))
	if err != nil {
		c.logger.Error("read avatar upload failed", "error", err)
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"success": false, "message": "unable to read avatar"})
		return
	}
	if int64(len(data)) > c.avatar.MaxBytes {
		c.writeError(ctx, domain.ErrAvatarTooLarge)
		return
	}
	contentType := strings.TrimSpace(strings.Split(header.Header.Get("Content-Type"), ";")[0])
	profile, err := c.usecase.UploadAvatar(ctx.Request.Context(), userID, contentType, data)
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, gin.H{"success": true, "message": "avatar uploaded", "data": profile})
}

func (c *ProfileController) DeleteAvatar(ctx *gin.Context) {
	userID, ok := subjectID(ctx)
	if !ok {
		return
	}
	profile, err := c.usecase.DeleteAvatar(ctx.Request.Context(), userID)
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, gin.H{"success": true, "message": "avatar deleted", "data": profile})
}

func (c *ProfileController) writeError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		ctx.JSON(nethttp.StatusNotFound, gin.H{"success": false, "message": "profile not found"})
	case errors.Is(err, domain.ErrInvalidInput), errors.Is(err, domain.ErrNoChanges), errors.Is(err, domain.ErrInvalidAvatar):
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrAvatarTooLarge):
		ctx.JSON(nethttp.StatusRequestEntityTooLarge, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrAvatarUnavailable):
		ctx.JSON(nethttp.StatusServiceUnavailable, gin.H{"success": false, "message": err.Error()})
	default:
		c.logger.Error("profile request failed", "error", err)
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"success": false, "message": "profile service request failed"})
	}
}

func (c *ProfileController) forbidden(ctx *gin.Context) {
	ctx.JSON(nethttp.StatusForbidden, gin.H{"success": false, "message": "required profile permission is missing"})
}

func subjectID(ctx *gin.Context) (uuid.UUID, bool) {
	claims, ok := ummuauth.ClaimsFromContext(ctx.Request.Context())
	if !ok || claims.PrincipalType != "user" {
		ctx.JSON(nethttp.StatusUnauthorized, gin.H{"success": false, "message": "authenticated subject is missing"})
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		ctx.JSON(nethttp.StatusUnauthorized, gin.H{"success": false, "message": "token subject is not a valid user ID"})
		return uuid.Nil, false
	}
	return userID, true
}

func pathUserID(ctx *gin.Context) (uuid.UUID, bool) {
	userID, err := uuid.Parse(ctx.Param("user_id"))
	if err != nil {
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"success": false, "message": "invalid user_id"})
		return uuid.Nil, false
	}
	return userID, true
}

func hasPermission(ctx *gin.Context, permission string) bool {
	claims, ok := ummuauth.ClaimsFromContext(ctx.Request.Context())
	return ok && claims.PrincipalType == "user" && claims.HasGlobalPermission(permission)
}
