package handlers

import (
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/services"
	"github.com/lomokwa/mc-manager/types"
)

// @Summary List backups
// @Description Lists all backup archives, newest first
// @Tags backups
// @Produce json
// @Success 200 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Security BearerAuth
// @Router /api/backups [get]
func ListBackupsHandler(c *gin.Context) {
	rt := runtimeFromRequest(c)
	backups, err := rt.ListBackups()
	if err != nil {
		slog.Error("failed to list backups", "err", err)
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: "failed to list backups"})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: backups})
}

// @Summary Create a backup
// @Description Creates a new backup of the world directory
// @Tags backups
// @Produce json
// @Success 201 {object} types.APIResponse
// @Failure 400 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Security BearerAuth
// @Router /api/backups [post]
func CreateBackupHandler(c *gin.Context) {
	slog.Debug("create backup request received")
	rt := runtimeFromRequest(c)

	info, err := rt.CreateBackup()
	if err != nil {
		slog.Error("failed to create backup", "err", err)
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: err.Error()})
		return
	}

	slog.Info("backup created", "backup", info.Name)
	c.JSON(http.StatusCreated, types.APIResponse{Success: true, Data: info})
}

// @Summary Delete a backup
// @Description Deletes a single backup archive by name
// @Tags backups
// @Produce json
// @Param name query string true "Backup name"
// @Success 200 {object} types.APIResponse
// @Failure 400 {object} types.APIResponse
// @Security BearerAuth
// @Router /api/backups [delete]
func DeleteBackupHandler(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: "name is required"})
		return
	}

	rt := runtimeFromRequest(c)
	if err := rt.DeleteBackup(name); err != nil {
		slog.Error("failed to delete backup", "backup", name, "err", err)
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: err.Error()})
		return
	}

	slog.Info("backup deleted", "backup", name)
	c.JSON(http.StatusOK, types.APIResponse{Success: true})
}

// @Summary Download a backup
// @Description Downloads a single backup archive by name
// @Tags backups
// @Produce application/zip
// @Param name query string true "Backup name"
// @Success 200 {file} binary
// @Failure 400 {object} types.APIResponse
// @Failure 404 {object} types.APIResponse
// @Security BearerAuth
// @Router /api/backups/download [get]
func DownloadBackupHandler(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: "name is required"})
		return
	}

	rt := runtimeFromRequest(c)
	path, err := rt.BackupFilePath(name)
	if err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{Error: err.Error()})
		return
	}

	c.FileAttachment(path, filepath.Base(path))
}

type restoreBackupRequest struct {
	Name string `json:"name" binding:"required"`
}

// @Summary Restore a backup
// @Description Restores the world directory from a backup archive. The server must be stopped first.
// @Tags backups
// @Accept json
// @Produce json
// @Param request body restoreBackupRequest true "Backup to restore"
// @Success 200 {object} types.APIResponse
// @Failure 400 {object} types.APIResponse
// @Security BearerAuth
// @Router /api/backups/restore [post]
func RestoreBackupHandler(c *gin.Context) {
	var req restoreBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: "invalid request body"})
		return
	}

	slog.Debug("restore backup request received", "backup", req.Name)

	rt := runtimeFromRequest(c)
	if err := rt.RestoreBackup(req.Name); err != nil {
		slog.Error("failed to restore backup", "backup", req.Name, "err", err)
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: err.Error()})
		return
	}

	slog.Warn("backup restored -- the live world was replaced", "backup", req.Name)
	c.JSON(http.StatusOK, types.APIResponse{Success: true})
}

// @Summary Get the backup schedule
// @Description Returns the current automatic backup configuration
// @Tags backups
// @Produce json
// @Success 200 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Security BearerAuth
// @Router /api/backups/config [get]
//
// GetBackupConfigHandler (and UpdateBackupConfigHandler below) deliberately
// do NOT resolve a runtime and stay on the services.LoadBackupConfig/
// SaveBackupConfig package functions: backup_config is still the single
// CHECK(id = 1) row it always was (see services/backup.go's comment on
// those two). They're still mounted under /api/servers/:sid/backups/config
// via ResolveServer -- so an unknown :sid 404s the same as every other
// namespaced route -- but a known :sid's config always reads/writes that
// same single global schedule, same as the flat route. Giving it a
// per-server schedule needs a server_id column, which is a real migration
// this PR intentionally doesn't take on (see PLAN-multi-server.md D1) --
// there's only ever one server able to reach this today anyway.
func GetBackupConfigHandler(c *gin.Context) {
	cfg, err := services.LoadBackupConfig()
	if err != nil {
		slog.Error("failed to load backup config", "err", err)
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: "failed to load backup config"})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: cfg})
}

// @Summary Update the backup schedule
// @Description Updates the automatic backup configuration and reloads the scheduler
// @Tags backups
// @Accept json
// @Produce json
// @Param request body types.BackupConfig true "Backup schedule"
// @Success 200 {object} types.APIResponse
// @Failure 400 {object} types.APIResponse
// @Failure 500 {object} types.APIResponse
// @Security BearerAuth
// @Router /api/backups/config [put]
func UpdateBackupConfigHandler(c *gin.Context) {
	var cfg types.BackupConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: "invalid request body"})
		return
	}

	if err := types.ValidateBackupConfig(cfg); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{Error: err.Error()})
		return
	}

	if err := services.SaveBackupConfig(cfg); err != nil {
		slog.Error("failed to save backup config", "err", err)
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: "failed to save backup config"})
		return
	}

	services.NotifyBackupConfigChanged()

	slog.Info("backup config updated", "enabled", cfg.Enabled, "interval_min", cfg.IntervalMinutes, "keep", cfg.Keep)
	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: cfg})
}
