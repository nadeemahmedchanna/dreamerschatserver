package main

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	rong "github.com/rongcloud/server-sdk-go/sdk"
	"github.com/spf13/viper"
)

const (
	CodeOK = iota
	CodeParamErr
	CodeIMServerErr
)

const portrait = "https://developer.rongcloud.cn/static/images/newversion-logo.png"

var roomMap sync.Map
var rongcloud *rong.RongCloud

type RoomDetail struct {
	Id     string `json:"roomId" binding:"required"`
	McuUrl string `json:"mcuUrl" binding:"required"`
	Name   string `json:"roomName" binding:"required"`
	UserId string `json:"pubUserId" binding:"required"`
	Date   int64  `json:"date"`
}

type Room struct {
	Id string `json:"roomId"`
}

type User struct {
	Id string `json:"id" binding:"required"`
}

func publish(c *gin.Context) {
	var room RoomDetail
	if err := c.ShouldBindJSON(&room); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": CodeParamErr, "desc": err.Error()})
		return
	}

	room.Date = time.Now().Local().UnixNano() / 1e6
	roomMap.Store(room.Id, room)
	c.JSON(http.StatusOK, gin.H{"code": CodeOK})
}

func unpublish(c *gin.Context) {
	var room Room
	if err := c.ShouldBindJSON(&room); err != nil || room.Id == "" {
		c.JSON(http.StatusOK, gin.H{"code": CodeParamErr, "desc": err.Error()})
		return
	}

	roomMap.Delete(room.Id)
	c.JSON(http.StatusOK, gin.H{"code": CodeOK})
}

func query(c *gin.Context) {
	var room Room
	if err := c.ShouldBindJSON(&room); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": CodeParamErr, "desc": err.Error()})
		return
	}

	roomList := make([]RoomDetail, 0)
	if room.Id == "" {
		roomMap.Range(func(k, v interface{}) bool {
			roomList = append(roomList, v.(RoomDetail))
			return true
		})
		sort.Slice(roomList, func(i, j int) bool {
			return roomList[i].Date > roomList[j].Date
		})
	} else if v, ok := roomMap.Load(room.Id); ok {
		roomList = append(roomList, v.(RoomDetail))
	}
	c.JSON(http.StatusOK, gin.H{"code": CodeOK, "roomList": roomList})
}

func getToken(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil || user.Id == "" {
		c.JSON(http.StatusOK, gin.H{"code": CodeParamErr, "desc": err.Error()})
		return
	}

	rUser, err := rongcloud.UserRegister(user.Id, user.Id, portrait)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": CodeIMServerErr, "desc": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "result": rUser})
}

func getAppVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "result": viper.GetStringMap("app.version")})
}

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %s", err))
	}

	rongcloud = rong.NewRongCloud(viper.GetString("rongcloud.appkey"), viper.GetString("rongcloud.secret"))

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(cors.Default())

	router.POST("/publish", publish)
	router.POST("/unpublish", unpublish)
	router.POST("/query", query)
	router.POST("/user/get_token", getToken)
	router.GET("/app/version", getAppVersion)

	router.Run(fmt.Sprintf(":%d", viper.GetInt("port")))
}
