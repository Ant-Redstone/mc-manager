package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/types"
)

func ListPlayersHandler(c *gin.Context) {
	log.Printf("list players request received")
	rt := runtimeFromRequest(c)
	players, err := rt.ListPlayers()
	if err != nil {
		log.Printf("failed to list players: %v", err)
		c.JSON(http.StatusInternalServerError, types.APIResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{Success: true, Data: players})
}
