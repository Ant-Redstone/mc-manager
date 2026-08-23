package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/middleware"
	"github.com/lomokwa/mc-manager/services"
	"github.com/lomokwa/mc-manager/types"
)

// UploadAvatarHandler stores a new profile picture for the caller --
// self-service, no admin.manage_users permission needed, same as
// UpdateProfileHandler. The request body is capped before the multipart form
// is even parsed, so an oversized upload is rejected without ever being
// buffered to disk or memory in full.
func UploadAvatarHandler(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{Error: "missing or invalid session"})
		return
	}

	// +1024 leaves headroom for multipart boundaries/field headers around the
	// actual file content, so a real AvatarMaxBytes-sized image isn't rejected
	// on packaging overhead alone.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, services.AvatarMaxBytes+1024)

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.JSON(http.StatusRequestEntityTooLarge, types.APIResponse{Success: false, Error: "image too large: max 5MB"})
			return
		}
		c.JSON(http.StatusBadRequest, types.APIResponse{Success: false, Error: "no image provided"})
		return
	}
	defer file.Close()

	if header.Size > services.AvatarMaxBytes {
		c.JSON(http.StatusRequestEntityTooLarge, types.APIResponse{Success: false, Error: "image too large: max 5MB"})
		return
	}

	if _, err := services.SaveAvatar(userID, file); err != nil {
		slog.Error("failed to save avatar", "err", err)
		c.JSON(http.StatusBadRequest, types.APIResponse{Success: false, Error: err.Error()})
		return
	}

	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{Error: "user not found"})
		return
	}
	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: user})
}

// DeleteAvatarHandler removes the caller's profile picture, if any.
func DeleteAvatarHandler(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, types.APIResponse{Error: "missing or invalid session"})
		return
	}

	if err := services.DeleteAvatar(userID); err != nil {
		slog.Error("failed to delete avatar", "err", err)
		c.JSON(http.StatusInternalServerError, types.APIResponse{Success: false, Error: "failed to delete avatar"})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{Success: true})
}
