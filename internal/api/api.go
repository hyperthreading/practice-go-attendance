package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AttendanceRecord struct {
	UserID string `json:"user_id"`
	Date   string `json:"date"`
	Time   string `json:"time"`
}

type SlackCommand struct {
	Token       string `form:"token" binding:"required"`
	TeamID      string `form:"team_id" binding:"required"`
	TeamDomain  string `form:"team_domain" binding:"required"`
	ChannelID   string `form:"channel_id" binding:"required"`
	ChannelName string `form:"channel_name" binding:"required"`
	UserID      string `form:"user_id" binding:"required"`
	UserName    string `form:"user_name" binding:"required"`
	Command     string `form:"command" binding:"required"`
	Text        string `form:"text" binding:"required"`
	ResponseURL string `form:"response_url" binding:"required"`
	TriggerID   string `form:"trigger_id" binding:"required"`
}

var database = map[string][]AttendanceRecord{}

func getToday() string {
	return "2023-01-01"
}

func New() *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/command/attendance", func(c *gin.Context) {
		data := SlackCommand{}
		c.Bind(&data)

		now := time.Now()
		today := now.Format("2006-01-02")
		time := now.Format("15:04:05")
		if _, ok := database[today]; !ok {
			database[today] = []AttendanceRecord{}
		}
		database[today] = append(database[today], AttendanceRecord{
			UserID: data.UserID,
			Date:   today,
			Time:   time,
		})

		response := gin.H{
			"message": "ok",
		}
		response["data"] = database

		c.JSON(http.StatusOK, response)
	})
	r.GET("/attendance/today", func(c *gin.Context) {
		today := time.Now().Format("2006-01-02")
		c.JSON(http.StatusOK, gin.H{
			"data": database[today],
		})
	})
	return r
}
