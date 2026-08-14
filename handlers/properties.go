package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/lomokwa/mc-manager/types"
)

func GetServerPropertiesHandler(c *gin.Context) {
	rt := runtimeFromRequest(c)
	props, err := rt.GetServerProperties()
	if err != nil {
		c.JSON(500, types.APIResponse{Error: "failed to read server properties"})
		return
	}

	c.JSON(200, types.APIResponse{Success: true, Data: props})
}

func UpdateServerPropertiesHandler(c *gin.Context) {
	rt := runtimeFromRequest(c)

	var req types.UpdateServerPropertiesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, types.APIResponse{Error: "invalid request body"})
		return
	}

	if err := types.ValidateServerProperties(req.Properties); err != nil {
		c.JSON(400, types.APIResponse{Error: err.Error()})
		return
	}

	err := rt.UpdateServerProperties(req.Properties)
	if err != nil {
		c.JSON(500, types.APIResponse{Error: "failed to update server properties"})
		return
	}

	c.JSON(200, types.APIResponse{Success: true})
}
