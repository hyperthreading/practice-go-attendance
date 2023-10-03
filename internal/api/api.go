package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AttendanceRecord struct {
	UserID     string `json:"userId"`
	UserName   string `json:"userName"`
	AttendedAt string `json:"attendedAt"`
	LeftAt     string `json:"leftAt,omitempty"`
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

func parseInformalTime(timeStr string, baseTime time.Time) (time.Time, error) {
	if len(timeStr) == 5 && timeStr[2] == ':' {
		timeStr = fmt.Sprintf("%vT%v:00%v", baseTime.Format("2006-01-02"), timeStr, baseTime.Format("-07:00"))
	} else if len(timeStr) == 4 && timeStr[1] == ':' {
		timeStr = fmt.Sprintf("%vT0%v:00%v", baseTime.Format("2006-01-02"), timeStr, baseTime.Format("-07:00"))
	} else {
		// 2. RFC3339 split by space and without timezone
		datetimeSplit := strings.Split(timeStr, " ")
		date := datetimeSplit[0]
		time := datetimeSplit[1]
		timeStr = fmt.Sprintf("%vT%v:00%v", date, time, baseTime.Format("-07:00"))
	}

	result, err := time.Parse(RFC3339_LONGFORM, timeStr)
	if err != nil {
		return time.Time{}, err
	}
	return result, nil
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

		spaceIndex := strings.Index(data.Text, " ")
		command := data.Text
		args := ""
		now := getTimeNow()
		targetTime := now
		if spaceIndex != -1 {
			command = command[:spaceIndex]
			args = data.Text[spaceIndex+1:]
		}

		if command == "attend" {

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

			records := attendanceRecordByUserId[userId]

			index := len(records)

			if index > 0 {
				lastAttendanceRecord := records[index-1]
				if lastAttendanceRecord.LeftAt == "" {
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
				AttendedAt: targetTime.Format(RFC3339_LONGFORM),
			}

			attendanceRecordByUserId[userId] = append(attendanceRecordByUserId[userId], attendanceRecord)

			response := gin.H{
				"message": "ok",
			}
			response["data"] = attendanceRecordByUserId

			c.JSON(http.StatusOK, response)
		} else if command == "leave" {
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

			if lastAttendanceRecord.LeftAt != "" {
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"message": "You have not attended yet",
					"code":    "not_attended_yet",
				})
				return
			}

			lastAttendanceRecord.LeftAt = targetTime.Format(RFC3339_LONGFORM)

			response := gin.H{
				"message": "ok",
			}
			response["data"] = attendanceRecordByUserId

			c.JSON(http.StatusOK, response)
		} else if command == "add" {
			var attendedAtStr, leavedAtStr string
			var attendedAt, leavedAt time.Time

			if args == "" {
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"message": "Invalid command",
					"code":    "invalid_command",
				})
				return
			}
			argsSplit := strings.Split(args, "~")
			if len(argsSplit) < 2 {
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"message": "Invalid command",
					"code":    "invalid_command",
				})
				return
			}
			attendedAtStr = strings.TrimSpace(argsSplit[0])
			leavedAtStr = strings.TrimSpace(argsSplit[1])

			attendedAt, err := parseInformalTime(attendedAtStr, now)
			if err != nil {
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"message": fmt.Sprintf("Invalid date format (%v)", err),
				})
				return
			}

			leavedAt, err = parseInformalTime(leavedAtStr, attendedAt)
			if err != nil {
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"message": fmt.Sprintf("Invalid date format (%v)", err),
				})
				return
			}

			userId := data.UserID
			records := attendanceRecordByUserId[userId]

			if records == nil {
				records = []AttendanceRecord{}
			}

			// Find the index to insert the new attendance record
			// The index is the first record that has attendedAt after the targetTime
			index := 0

			if len(records) > 0 {
				for i, attendanceRecord := range records {
					rcdAttendedAt, err := time.Parse(RFC3339_LONGFORM, attendanceRecord.AttendedAt)

					if err != nil {
						c.JSON(http.StatusUnprocessableEntity, gin.H{
							"message": fmt.Sprintf("Invalid date format (%v)", err),
						})
						return
					}
					if rcdAttendedAt.Compare(attendedAt) >= 0 {
						index = i
						break
					} else if i == len(records)-1 {
						index = len(records)
					}
				}

				// check whether target time is between the previous record and the current record
				if index > 0 {
					prevAttendanceRecord := records[index-1]
					if prevAttendanceRecord.LeftAt == "" {
						c.JSON(http.StatusUnprocessableEntity, gin.H{
							"message": "You can not attend while you are attending",
							"code":    "invalid_date",
						})
						return
					}

					prevLeftAt, err := time.Parse(RFC3339_LONGFORM, prevAttendanceRecord.LeftAt)
					if err != nil {
						c.JSON(http.StatusUnprocessableEntity, gin.H{
							"message": fmt.Sprintf("Invalid date format (%v)", err),
						})
						return
					}
					if prevLeftAt.Compare(attendedAt) > 0 {
						c.JSON(http.StatusUnprocessableEntity, gin.H{
							"message": "You can not attend while you are attending",
							"code":    "invalid_date",
						})
						return
					}
				}
			}

			// insert the record between the previous record and the current record
			attendanceRecord := AttendanceRecord{
				UserID:     data.UserID,
				UserName:   data.UserName,
				AttendedAt: attendedAt.Format(RFC3339_LONGFORM),
				LeftAt:     leavedAt.Format(RFC3339_LONGFORM),
			}

			var newRecords []AttendanceRecord
			if len(records) == 0 {
				newRecords = append(records, attendanceRecord)
			} else {
				newRecords = append(records[:index+1], records[index:]...)
				newRecords[index] = attendanceRecord
			}
			attendanceRecordByUserId[userId] = newRecords

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
