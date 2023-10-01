package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AttendanceRecord struct {
	UserID     string `json:"user_id"`
	AttendedAt string `json:"attended_at"`
	LeavedAt   string `json:"leaved_at"`
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

var ISO8601 = "2006-01-02T15:04:05Z07:00"

var attendanceRecordByUserId = map[string][]AttendanceRecord{}

var fixedTime time.Time

func getTimeNow() time.Time {
	if !fixedTime.IsZero() {
		return time.Now()
	}
	return fixedTime
}

func fixTimeNow(t time.Time) {
	fixedTime = t
}

func resetTimeNow() {
	fixedTime = time.Time{}
}

func New() *gin.Engine {
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	r.POST("/test/fix-time", func(c *gin.Context) {
		data := struct {
			Time string `json:"time"`
		}{}
		c.Bind(&data)

		timeNow, err := time.Parse(ISO8601, data.Time)

		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": fmt.Sprintf("Invalid time format (%v)", err),
			})
			return
		}

		fixTimeNow(timeNow)

		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	r.POST("/test/reset-time", func(c *gin.Context) {
		resetTimeNow()
		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	r.POST("/command/attend", func(c *gin.Context) {
		data := SlackCommand{}
		c.Bind(&data)

		if data.Text == "attend" {

			userId := data.UserID

			if userId == "" {
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"message": "Invalid user id",
					"code":    "invalid_user_id",
				})
				return
			}

			if attendanceRecordByUserId[userId] == nil {
				attendanceRecordByUserId[userId] = []AttendanceRecord{}
			}

			if len(attendanceRecordByUserId[userId]) > 0 {
				lastAttendanceRecord := attendanceRecordByUserId[userId][len(attendanceRecordByUserId[userId])-1]
				if lastAttendanceRecord.LeavedAt == "" {
					c.JSON(http.StatusUnprocessableEntity, gin.H{
						"message": "You have not leaved yet",
						"code":    "not_leaved_yet",
					})
					return
				}
			}

			attendanceRecord := AttendanceRecord{
				UserID:     data.UserID,
				AttendedAt: getTimeNow().Format(ISO8601),
			}

			attendanceRecordByUserId[userId] = append(attendanceRecordByUserId[userId], attendanceRecord)

			response := gin.H{
				"message": "ok",
			}
			response["data"] = attendanceRecordByUserId

			c.JSON(http.StatusOK, response)
		} else if data.Text == "leave" {
			userId := data.UserID
			if userId == "" {
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"message": "Invalid user id",
					"code":    "invalid_user_id",
				})
				return
			}

			if attendanceRecordByUserId[userId] == nil || len(attendanceRecordByUserId[userId]) == 0 {
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"message": "You have not attended yet",
					"code":    "not_attended_yet",
				})
				return
			}

			lastAttendanceRecord := attendanceRecordByUserId[userId][len(attendanceRecordByUserId[userId])-1]

			if lastAttendanceRecord.LeavedAt != "" {
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"message": "You have not attended yet",
					"code":    "not_attended_yet",
				})
				return
			}

			lastAttendanceRecord.LeavedAt = getTimeNow().Format(ISO8601)

			response := gin.H{
				"message": "ok",
			}
			response["data"] = attendanceRecordByUserId

			c.JSON(http.StatusOK, response)
		} else {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": "Invalid command",
				"code":    "invalid_command",
			})
		}

	})

	r.GET("/attendance/attended", func(c *gin.Context) {
		today := time.Now().Format("2006-01-02")
		c.JSON(http.StatusOK, gin.H{
			"data": attendanceRecordByUserId[today],
		})
	})
	return r
}
