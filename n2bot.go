// n2irc receives posted JSON and renders associated
// templates to an IRC channel
//
// txn2.com
package main

import (
	"io/ioutil"
	"os"
	"time"

	"crypto/tls"

	"strings"

	"github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/thoj/go-ircevent"
	"github.com/txn2/n2bot/pkg"
	"github.com/txn2/service/ginack"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

func main() {
	// Default and consistent environment variables
	// help standardize k8s configs and documentation
	//
	port := getEnv("PORT", "8080")
	debug := getEnv("DEBUG", "false")
	cfgFile := getEnv("CONFIG", "")
	basePath := getEnv("BASE_PATH", "")
	server := getEnv("SERVER", "irc.freenode.net:7000")
	ssl := getEnv("SSL", "true")
	channel := getEnv("CHANNEL", "##n2bot,##n2bot2")
	nick := getEnv("NICK", "n2bot")
	token := getEnv("TOKEN", "abc")

	channels := strings.Split(channel, ",")

	// load a configuration yml if one is specified
	//
	cfg := pkg.Configuration{}
	if cfgFile != "" {
		ymlData, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			panic(err)
		}

		err = yaml.Unmarshal([]byte(ymlData), &cfg)
		if err != nil {
			panic(err)
		}
	}

	// parse templates and compile regexes
	err := cfg.Warmup()
	if err != nil {
		panic(err)
	}

	gin.SetMode(gin.ReleaseMode)

	if debug == "true" {
		gin.SetMode(gin.DebugMode)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err.Error())
	}

	if debug == "true" {
		logger, _ = zap.NewDevelopment()
	}

	ircConn := irc.IRC(nick, nick)

	handler := pkg.Handler{
		Logger: logger,
		Cfg:    cfg,
		IRC:    ircConn,
		Token:  token,
	}

	// fire up IRC client
	go func() {

		ircConn.VerboseCallbackHandler = true

		if ssl == "true" {
			ircConn.UseTLS = true
			ircConn.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		}

		ircConn.AddCallback("001", func(e *irc.Event) {
			logger.Debug("Joining channel...")
			for _, ch := range channels {
				ircConn.Join(ch)
			}
		})

		err := ircConn.Connect(server)
		if err != nil {
			logger.Error("Err %s", zap.Error(err))
			return
		}
		ircConn.Loop()
	}()

	// router
	r := gin.New()
	rg := r.Group(basePath)

	// HTTP middleware
	//
	rg.Use(ginzap.Ginzap(logger, time.RFC3339, true))

	// routes
	//
	rg.GET("/",
		func(c *gin.Context) {

			// call external libs for business logic here

			ack := ginack.Ack(c)
			ack.SetPayload(gin.H{"message": "welcome"})

			// return
			c.JSON(ack.ServerCode, ack)
			return
		},
	)

	// webhook
	rg.POST("/in/:producer/:channels/:token", handler.MessageHandler)

	// for external status check
	r.GET(basePath+"/status",
		func(c *gin.Context) {
			ack := ginack.Ack(c)
			p := gin.H{"message": "alive"}

			if c.Query("noack") == "true" {
				c.JSON(200, p)
				return
			}

			ack.SetPayload(p)
			c.JSON(ack.ServerCode, ack)
		},
	)

	// default no route
	r.NoRoute(func(c *gin.Context) {
		ack := ginack.Ack(c)
		ack.SetPayload(gin.H{"message": "not found"})
		ack.ServerCode = 404
		ack.Success = false

		// return
		c.JSON(ack.ServerCode, ack)
	})

	r.Run(":" + port)
}

// getEnv gets an environment variable or sets a default if
// one does not exist.
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}

	return value
}
