package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/services"
	"github.com/lomokwa/mc-manager/types"
	"github.com/lomokwa/mc-manager/utils"
)

// @Summary Start the Minecraft server
// @Description Downloads the server jar if needed, prepares server files, and starts the process
// @Tags server
// @Accept json
// @Produce json
// @Param request body types.StartServerRequest true "Server configuration"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/start [post]
func StartServerHandler(c *gin.Context) {
	log.Printf("start request received")

	var req types.StartServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if !utils.FileExists("./minecraft-server/server.jar") {
		log.Printf("server.jar not found, downloading latest")
		err := services.DownloadLatestServerJar("./minecraft-server/server.jar", req.ReleaseVersion)
		if err != nil {
			log.Printf("failed to download server.jar: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		log.Printf("server.jar downloaded successfully")
	}

	log.Printf("creating server files")
	if err := services.PrepareServerFiles("./minecraft-server", req.CreateLaunchScript, req.ConfigureProperties, req.Properties); err != nil {
		log.Printf("failed to prepare server files: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("starting server process")
	output, err := services.StartServerProcess()
	if err != nil {
		log.Printf("failed to start server process: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("server process started")
	c.JSON(http.StatusOK, gin.H{"success": true, "output": output})
}

// @Summary Stop the Minecraft server
// @Description Stops the running Minecraft server process
// @Tags server
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /api/stop [post]
func StopServerHandler(c *gin.Context) {
	log.Printf("stop request received")

	output, err := services.StopServerProcess()
	if err != nil {
		log.Printf("failed to stop server process: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("server process stopped")
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// @Summary Get server status
// @Description Returns whether the Minecraft server is currently running
// @Tags server
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/status [get]
func StatusHandler(c *gin.Context) {
	log.Printf("status request received")
	c.JSON(http.StatusOK, gin.H{"running": services.IsServerRunning()})
}
