package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/types"
)

func ListPlayersHandler(c *gin.Context) {
	slog.Debug("list players request received")
	rt := runtimeFromRequest(c)
	players, err := rt.ListPlayers()
	if err != nil {
		slog.Error("failed to list players", "err", err)
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: players})
}

func DeletePlayersHandler(c *gin.Context) {
	slog.Debug("delete player request received")

	uuid := c.Param("uuid")

	rt := runtimeFromRequest(c)
	player, err := rt.DeletePlayer(uuid)
	if err != nil {
		slog.Error("failed to delete player", "err", err)
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: player})
}
