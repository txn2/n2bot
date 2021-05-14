package main

import (
	"crypto/tls"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	irc "github.com/thoj/go-ircevent"
	"github.com/txn2/n2bot/pkg"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var Version = "0.0.0"

var (
	ipEnv       = getEnv("IP", "127.0.0.1")
	portEnv     = getEnv("PORT", "8080")
	debugEnv    = getEnv("DEBUG", "false")
	cfgFileEnv  = getEnv("CONFIG", "./example.yml")
	basePathEnv = getEnv("BASE_PATH", "")
	serverEnv   = getEnv("SERVER", "irc.freenode.net:7000")
	sslEnv      = getEnv("SSL", "true")
	channelEnv  = getEnv("CHANNEL", "##n2bot,##n2bot2")
	nickEnv     = getEnv("NICK", "n2bot")
	tokenEnv    = getEnv("TOKEN", "abc")
)

func main() {

	var (
		ip       = flag.String("ip", ipEnv, "Server IP address to bind to.")
		port     = flag.String("port", portEnv, "Server port.")
		debug    = flag.String("debug", debugEnv, "debug mode.")
		cfgFile  = flag.String("cfgFile", cfgFileEnv, "path to config file")
		basePath = flag.String("basePath", basePathEnv, "base path")
		server   = flag.String("server", serverEnv, "IRC server")
		ssl      = flag.String("ssl", sslEnv, "ssl")
		channel  = flag.String("channel", channelEnv, "channels")
		nick     = flag.String("nick", nickEnv, "nick")
		token    = flag.String("token", tokenEnv, "token")
	)
	flag.Parse()

	channels := strings.Split(*channel, ",")

	// load a configuration yml if one is specified
	//
	cfg := pkg.Configuration{}
	if *cfgFile != "" {
		ymlData, err := ioutil.ReadFile(*cfgFile)
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

	if *debug == "true" {
		gin.SetMode(gin.DebugMode)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err.Error())
	}

	if *debug == "true" {
		logger, _ = zap.NewDevelopment()
	}

	ircConn := irc.IRC(*nick, *nick)

	handler := pkg.Handler{
		Logger: logger,
		Cfg:    cfg,
		IRC:    ircConn,
		Token:  *token,
		Cache:  cache.New(1*time.Minute, 5*time.Minute),
	}

	// fire up IRC client
	go func() {

		ircConn.VerboseCallbackHandler = true

		if *ssl == "true" {
			ircConn.UseTLS = true
			ircConn.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		}

		ircConn.AddCallback("001", func(e *irc.Event) {
			logger.Debug("Joining channel...")
			for _, ch := range channels {
				ircConn.Join(ch)
			}
		})

		err := ircConn.Connect(*server)
		if err != nil {
			logger.Error("Err %s", zap.Error(err))
			return
		}
		ircConn.Loop()
	}()

	// router
	r := gin.New()
	rg := r.Group(*basePath)

	// HTTP middleware
	//
	rg.Use(ginzap.Ginzap(logger, time.RFC3339, true))

	// routes
	//
	rg.GET("/",
		func(c *gin.Context) {
			// return
			c.JSON(http.StatusOK, gin.H{"message": "welcome", "version": Version})
			return
		},
	)

	// webhook
	rg.POST("/in/:producer/:channels/:token", handler.MessageHandler)

	// for external status check
	r.GET(*basePath+"/status",
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "alive", "version": Version})
		},
	)

	// default no route
	r.NoRoute(func(c *gin.Context) {
		// return
		c.JSON(http.StatusNotFound, gin.H{"message": "not found"})
	})

	err = r.Run(*ip + ":" + *port)
	if err != nil {
		panic(err)
	}
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
