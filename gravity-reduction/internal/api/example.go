package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gravity-reduction/internal/gravity"
)

// handleExample 返回内置示例测点的归算结果，便于调用方手工验算。
func handleExample(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"description": "内置示例：φ=40°N, h=500m, ρ=2.67 g/cm^3, gobs=9.802640 m/s^2",
		"request":     gravity.ExampleObservation,
		"result":      gravity.ReducePoint(gravity.ExampleObservation),
	})
}
