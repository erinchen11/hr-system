package handlers

import (
	"net/http"
	"time" 

	common "github.com/erinchen11/hr-system/internal/models/common"
	"github.com/gin-gonic/gin"
)

// CheckLiveHandler 結構體 (不需要欄位)
type CheckLiveHandler struct{}

// NewCheckLiveHandler 構造函數
func NewCheckLiveHandler() *CheckLiveHandler {
	return &CheckLiveHandler{}
}

func (h *CheckLiveHandler) CheckLive(c *gin.Context) {
	c.JSON(http.StatusOK, common.Response{
		Code:    http.StatusOK,
		Message: "Server is alive and kicking! (Struct Method)",
		Data: gin.H{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
}
