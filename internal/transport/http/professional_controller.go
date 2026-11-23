package http

import (
	"errors"
	"log/slog"
	nethttp "net/http"

	ummuauth "github.com/AgungDev/ummu-go-auth"
	"github.com/AgungDev/ummu-profile-service/internal/domain"
	"github.com/AgungDev/ummu-profile-service/internal/domain/dto"
	"github.com/AgungDev/ummu-profile-service/internal/usecases"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProfessionalController struct {
	usecase usecases.ProfessionalUsecase
	auth    *ummuauth.Auth
	logger  *slog.Logger
}

func NewProfessionalController(
	usecase usecases.ProfessionalUsecase,
	auth *ummuauth.Auth,
	logger *slog.Logger,
) *ProfessionalController {
	return &ProfessionalController{usecase: usecase, auth: auth, logger: logger}
}

func (c *ProfessionalController) Register(router *gin.Engine) {
	protected := router.Group("/api/v1/profiles")
	protected.Use(JWTMiddleware(c.auth))
	for _, resource := range []string{"students", "lecturers", "employees"} {
		protected.GET("/me/"+resource, c.getMine(resource))
		protected.PUT("/me/"+resource, LimitRequestBody(1<<20), c.putMine(resource))
		protected.DELETE("/me/"+resource, c.deleteMine(resource))
		protected.GET("/:user_id/"+resource, c.getForUser(resource))
		protected.PUT("/:user_id/"+resource, LimitRequestBody(1<<20), c.putForUser(resource))
		protected.DELETE("/:user_id/"+resource, c.deleteForUser(resource))
	}
}

func (c *ProfessionalController) getMine(resource string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, ok := subjectID(ctx)
		if !ok {
			return
		}
		c.get(ctx, resource, userID)
	}
}

func (c *ProfessionalController) putMine(resource string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, ok := subjectID(ctx)
		if !ok {
			return
		}
		c.put(ctx, resource, userID, false)
	}
}

func (c *ProfessionalController) deleteMine(resource string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userID, ok := subjectID(ctx)
		if !ok {
			return
		}
		c.delete(ctx, resource, userID)
	}
}

func (c *ProfessionalController) getForUser(resource string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !hasPermission(ctx, "profile.admin:read") {
			c.forbidden(ctx)
			return
		}
		userID, ok := pathUserID(ctx)
		if !ok {
			return
		}
		c.get(ctx, resource, userID)
	}
}

func (c *ProfessionalController) putForUser(resource string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !hasPermission(ctx, "profile.admin:update") {
			c.forbidden(ctx)
			return
		}
		userID, ok := pathUserID(ctx)
		if !ok {
			return
		}
		c.put(ctx, resource, userID, true)
	}
}

func (c *ProfessionalController) deleteForUser(resource string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !hasPermission(ctx, "profile.admin:update") {
			c.forbidden(ctx)
			return
		}
		userID, ok := pathUserID(ctx)
		if !ok {
			return
		}
		c.delete(ctx, resource, userID)
	}
}

func (c *ProfessionalController) get(ctx *gin.Context, resource string, userID uuid.UUID) {
	var data interface{}
	var err error
	switch resource {
	case "students":
		data, err = c.usecase.GetStudent(ctx.Request.Context(), userID)
	case "lecturers":
		data, err = c.usecase.GetLecturer(ctx.Request.Context(), userID)
	case "employees":
		data, err = c.usecase.GetEmployee(ctx.Request.Context(), userID)
	}
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, gin.H{"success": true, "data": data})
}

func (c *ProfessionalController) put(ctx *gin.Context, resource string, userID uuid.UUID, admin bool) {
	var data interface{}
	var err error
	switch resource {
	case "students":
		var req dto.StudentRequest
		if !c.bind(ctx, &req) {
			return
		}
		if admin {
			data, err = c.usecase.SaveStudentForUser(ctx.Request.Context(), userID, req)
		} else {
			data, err = c.usecase.SaveMyStudent(ctx.Request.Context(), userID, req)
		}
	case "lecturers":
		var req dto.LecturerRequest
		if !c.bind(ctx, &req) {
			return
		}
		if admin {
			data, err = c.usecase.SaveLecturerForUser(ctx.Request.Context(), userID, req)
		} else {
			data, err = c.usecase.SaveMyLecturer(ctx.Request.Context(), userID, req)
		}
	case "employees":
		var req dto.EmployeeRequest
		if !c.bind(ctx, &req) {
			return
		}
		if admin {
			data, err = c.usecase.SaveEmployeeForUser(ctx.Request.Context(), userID, req)
		} else {
			data, err = c.usecase.SaveMyEmployee(ctx.Request.Context(), userID, req)
		}
	}
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, gin.H{"success": true, "message": resource + " saved", "data": data})
}

func (c *ProfessionalController) delete(ctx *gin.Context, resource string, userID uuid.UUID) {
	var err error
	switch resource {
	case "students":
		err = c.usecase.DeleteStudent(ctx.Request.Context(), userID)
	case "lecturers":
		err = c.usecase.DeleteLecturer(ctx.Request.Context(), userID)
	case "employees":
		err = c.usecase.DeleteEmployee(ctx.Request.Context(), userID)
	}
	if err != nil {
		c.writeError(ctx, err)
		return
	}
	ctx.JSON(nethttp.StatusOK, gin.H{"success": true, "message": resource + " deleted"})
}

func (c *ProfessionalController) bind(ctx *gin.Context, target interface{}) bool {
	if err := ctx.ShouldBindJSON(target); err != nil {
		var maxBytesError *nethttp.MaxBytesError
		if errors.As(err, &maxBytesError) {
			ctx.JSON(nethttp.StatusRequestEntityTooLarge, gin.H{"success": false, "message": "request body too large"})
			return false
		}
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"success": false, "message": "invalid professional request"})
		return false
	}
	return true
}

func (c *ProfessionalController) writeError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrProfessionalNotFound), errors.Is(err, domain.ErrNotFound):
		ctx.JSON(nethttp.StatusNotFound, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrInvalidInput):
		ctx.JSON(nethttp.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, domain.ErrConflict):
		ctx.JSON(nethttp.StatusConflict, gin.H{"success": false, "message": err.Error()})
	default:
		c.logger.Error("professional profile request failed", "error", err)
		ctx.JSON(nethttp.StatusInternalServerError, gin.H{"success": false, "message": "profile service request failed"})
	}
}

func (c *ProfessionalController) forbidden(ctx *gin.Context) {
	ctx.JSON(nethttp.StatusForbidden, gin.H{"success": false, "message": "required profile permission is missing"})
}
