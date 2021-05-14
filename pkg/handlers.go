package pkg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	irc "github.com/thoj/go-ircevent"
	"go.uber.org/zap"
)

type MsgIn map[string]interface{}

type Handler struct {
	Logger *zap.Logger
	Cfg    Configuration
	IRC    *irc.Connection
	Token  string
	Cache  *cache.Cache
}

func (h *Handler) MessageHandler(c *gin.Context) {
	producer := c.Param("producer")
	channelStr := c.Param("channels")
	token := c.Param("token")

	channels := strings.Split(channelStr, ",")

	if token != h.Token {
		c.JSON(http.StatusUnauthorized, gin.H{"status": "fail", "message": "bad token"})
		return
	}

	h.Logger.Info("Webhook",
		zap.String("type", "webhook"),
		zap.String("producer", producer),
	)

	rs, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "fail", "error": err.Error()})
		return
	}

	msgOut := ""

	// inbound JSON
	msgIn := MsgIn{}

	err = json.Unmarshal(rs, &msgIn)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "fail", "error": err.Error()})
		return
	}

	h.Logger.Debug("Inbound Json",
		zap.String("type", "inboundJson"),
		zap.String("json", string(rs)),
	)

	// get template for producer
	tmpl, err := h.templateProducer(producer, msgIn)
	if err != nil {
		h.Logger.Info(err.Error(),
			zap.String("type", "templateNotFound"),
			zap.String("producer", producer),
		)

		c.JSON(http.StatusBadRequest, gin.H{"status": "ok", "state": "no template found", "msg_out": ""})
		return
	}

	if tmpl != nil {
		var tplReturn bytes.Buffer
		if err := tmpl.Execute(&tplReturn, msgIn); err != nil {
			h.Logger.Error("Template executions failed: " + err.Error())
		}

		msgOut = tplReturn.String()
	}

	// regex replacements
	for rgx, replace := range h.Cfg.CompiledRegexes {
		msgOut = rgx.ReplaceAllString(msgOut, replace)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "msg_out": msgOut})

	// post message to specified channels
	if msgOut != "" {

		// final cleanup
		msgOut = strings.Join(strings.Fields(msgOut), " ")

		// prevent duplicate message within time span
		// this helps prevent conditions where similar posts produce the
		// exact same message
		_, found := h.Cache.Get(msgOut)
		if !found {
			for _, ch := range channels {
				h.IRC.SendRaw(fmt.Sprintf("PRIVMSG %s : %s", strings.Replace(ch, "^", "#", -1), msgOut))
			}
			// add message to cache
			h.Cache.Set(msgOut, "sent", cache.DefaultExpiration)

			return
		}

		h.Logger.Info("CACHE HIT", zap.String("msgOut", msgOut))
	}

	return
}

func (h *Handler) templateProducer(producer string, msgIn MsgIn) (*template.Template, error) {
	// find a template for a producer and any content rules
	for p, rt := range h.Cfg.ParsedTemplates {
		if p == producer {
			h.Logger.Debug("Checking producer: " + p)
			for cr, t := range rt {
				// found empty content rule, so we match and return
				if cr.Key == "" && cr.Equals == "" {
					h.Logger.Debug("Empty content rule.")
					return t, nil
				}

				if msgIn[cr.Key] == cr.Equals {
					h.Logger.Debug("FOUND content rule.")
					return t, nil
				}
			}
		}
	}

	return nil, errors.New("no template found")
}
