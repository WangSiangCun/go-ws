package engine

import (
	"go-ws/config"
)

type Message struct {
	Message   string   `json:"message"`
	TargetIds []string `json:"target_ids"`
	SourceId  string   `json:"source_id"`
}
type Engine struct {
	Config          *config.Config
	IsOpenJWT       bool
	SendHandlers    []func(msg *Message) *Message
	ReceiveHandlers []func(msg *Message) *Message
}

func NewEngine(config *config.Config) *Engine {
	return &Engine{Config: config}
}
func (e *Engine) OpenJWT(jwt config.JWT) {
	e.IsOpenJWT = true
}
func (e *Engine) SetSendHandlers(SendHandlers []func(message *Message) *Message) {
	e.SendHandlers = SendHandlers
}
func (e *Engine) SetReadHandlers(ReadHandlers []func(message *Message) *Message) {
	e.ReceiveHandlers = ReadHandlers
}
