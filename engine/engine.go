package engine

import (
	"go-ws/config"
	"go-ws/wsContext"
)

type Message struct {
	Message   string   `json:"message"`
	TargetIds []string `json:"target_ids"`
	SourceId  string   `json:"source_id"`
}
type Engine struct {
	Config               *config.Config
	IsOpenJWT            bool
	SendHandlers         []HandlersFunc
	ReceiveHandlers      []HandlersFunc
	IsServerHandlerModel bool
}
type HandlersFunc func(e *Engine, wsCtx wsContext.WSContext, message *Message) *Message

func NewEngine(config *config.Config) *Engine {
	return &Engine{Config: config}
}
func (e *Engine) OpenJWT(jwt config.JWT) {
	e.IsOpenJWT = true
}
func (e *Engine) SetSendHandlers(SendHandlers []HandlersFunc) {
	e.SendHandlers = SendHandlers
}
func (e *Engine) SetReadHandlers(ReadHandlers []HandlersFunc) {
	e.ReceiveHandlers = ReadHandlers
}
