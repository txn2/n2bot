package pkg

import (
	"bytes"
	"encoding/json"
	"text/template"

	"errors"

	"fmt"

	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thoj/go-ircevent"
	"github.com/txn2/service/ginack"
	"go.uber.org/zap"
)

type MsgIn map[string]interface{}

type Handler struct {
	Logger *zap.Logger
	Cfg    Configuration
	IRC    *irc.Connection
	Token  string
}

func (h *Handler) MessageHandler(c *gin.Context) {
	ack := ginack.Ack(c)

	producer := c.Param("producer")
	channelStr := c.Param("channels")
	token := c.Param("token")

	channels := strings.Split(channelStr, ",")

	if token != h.Token {
		ack.ServerCode = 401
		ack.SetPayload(gin.H{"status": "fail", "message": "bad token"})
		ack.Success = false
		c.JSON(ack.ServerCode, ack)
		return
	}

	h.Logger.Info("Webhook",
		zap.String("type", "webhook"),
		zap.String("producer", producer),
	)

	rs, err := c.GetRawData()
	if err != nil {
		ack.ServerCode = 500
		ack.SetPayload(gin.H{"status": "fail", "error": err.Error()})
		c.JSON(ack.ServerCode, ack)
		return
	}

	msgOut := ""

	// inbound JSON
	msgIn := MsgIn{}

	err = json.Unmarshal(rs, &msgIn)
	if err != nil {
		ack.ServerCode = 500
		ack.SetPayload(gin.H{"status": "fail", "error": err.Error()})
		c.JSON(ack.ServerCode, ack)
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

		ack.SetPayload(gin.H{"status": "ok", "state": "no template found", "msg_out": ""})
		c.JSON(ack.ServerCode, ack)
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

	ack.SetPayload(gin.H{"status": "ok", "msg_out": msgOut})
	c.JSON(ack.ServerCode, ack)

	// post message to specified channels
	if msgOut != "" {
		for _, ch := range channels {
			h.IRC.SendRaw(fmt.Sprintf("PRIVMSG %s : %s", strings.Replace(ch, "^", "#", -1), msgOut))
		}
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
