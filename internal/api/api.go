package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AttendanceRecord struct {
	UserID     string `json:"userId"`
	UserName   string `json:"userName"`
	AttendedAt string `json:"attendedAt"`
	LeavedAt   string `json:"leavedAt,omitempty"`
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

var RFC3339_LONGFORM = "2006-01-02T15:04:05Z07:00"

var attendanceRecordByUserId = map[string][]AttendanceRecord{}

var fixedTime time.Time

func getTimeNow() time.Time {
	if fixedTime.IsZero() {
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

		timeNow, err := time.Parse(RFC3339_LONGFORM, data.Time)

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

	r.POST("/test/reset-database", func(c *gin.Context) {
		attendanceRecordByUserId = map[string][]AttendanceRecord{}
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
				UserName:   data.UserName,
				AttendedAt: getTimeNow().Format(RFC3339_LONGFORM),
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

			lastAttendanceRecord.LeavedAt = getTimeNow().Format(RFC3339_LONGFORM)

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

	r.GET("/user_list/attended", func(c *gin.Context) {
		params := c.Request.URL.Query()
		date := params.Get("date")
		if date == "" {
			date = getTimeNow().Format("2006-01-02")
		}
		tz := params.Get("tz")
		if tz == "" {
			tz = "KST"
		}
		queryDateTime, err := time.Parse("2006-01-02 MST", date+" "+tz)
		if err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"message": fmt.Sprintf("Invalid date format (%v)", err),
			})
			return
		}

		attended_user_list := []AttendanceRecord{}

		for _, attendanceRecords := range attendanceRecordByUserId {
			for _, attendanceRecord := range attendanceRecords {
				attendedAt, err := time.Parse(RFC3339_LONGFORM, attendanceRecord.AttendedAt)
				if err != nil {
					c.JSON(http.StatusUnprocessableEntity, gin.H{
						"message": fmt.Sprintf("Invalid date format (%v)", err),
					})
					return
				}
				if attendedAt.Compare(queryDateTime) >= 0 {
					attendanceRecord.AttendedAt = attendedAt.Format(RFC3339_LONGFORM)
					attended_user_list = append(attended_user_list, attendanceRecord)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "ok",
			"data":    attended_user_list,
		})
	})
	return r
}
