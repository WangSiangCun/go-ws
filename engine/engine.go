package engine

import (
	"github.com/WangSiangCun/go-ws/config"
	"github.com/WangSiangCun/go-ws/wsContext"
	"reflect"
)

type Message struct {
	Message   string   `json:"message"`
	TargetIds []string `json:"target_ids"`
	SourceId  string   `json:"source_id"`
	Type      int64    `json:"type"`
}
type Engine struct {
	Config               *config.Config
	IsOpenJWT            bool
	SendHandlers         []HandlersFunc
	SendParameters       [][]any
	ReceiveHandlers      []HandlersFunc
	ReceiveParameters    [][]any
	IsServerHandlerModel bool
}
type HandlersFunc func(e *Engine, wsCtx wsContext.WSContext, message *Message, opts ...any) (res *Message)

func NewEngine(config *config.Config) *Engine {
	return &Engine{Config: config}
}
func (e *Engine) OpenJWT(jwt config.JWT) {
	e.IsOpenJWT = true
}
func (e *Engine) SetSendHandlers(SendHandlers []HandlersFunc) {
	e.SendHandlers = SendHandlers
}
func (e *Engine) SetSendParameters(opts [][]any) {
	e.SendParameters = opts
}

func (e *Engine) SetReceiverHandlers(ReadHandlers []HandlersFunc) {
	e.ReceiveHandlers = ReadHandlers
}
func (e *Engine) SetReceiverParameters(opts [][]any) {
	e.ReceiveParameters = opts
}

// RunHandlers 执行插件
func (e *Engine) runHandlers(wsCtx wsContext.WSContext, handlers []HandlersFunc, parametersPointer *[][]any, message *Message) {
	parameters := *parametersPointer
	if len(handlers) != len(parameters) {
		panic("handlers and parameters length not equal")
	}
	args := []reflect.Value{reflect.ValueOf(e), reflect.ValueOf(wsCtx), reflect.ValueOf(message)}

	for i, handler := range handlers {
		// Get the number of optional parameters from the handler function type
		handlerType := reflect.TypeOf(handler)
		numOptParams := handlerType.NumIn() - 3 // Subtract 3 for e, wsCtx, message params

		// Add optional parameters
		for j := 0; j < numOptParams; j++ {
			// 如果args容量不够就扩大
			if len(args)-3 <= j {
				args = append(args, reflect.ValueOf(parameters[i][j]))
			} else {
				args[j+3] = reflect.ValueOf(parameters[i][j])
			}
		}

		// Call the handler with the prepared arguments
		result := reflect.ValueOf(handler).Call(args)

		// If the handler returns a message, use it for the next handler
		if len(result) > 0 && !result[0].IsNil() {
			message = result[0].Interface().(*Message)
		}
	}
}

// RunSendHandlers 执行插件
func (e *Engine) RunSendHandlers(wsCtx wsContext.WSContext, message *Message) {
	e.runHandlers(wsCtx, e.SendHandlers, &e.SendParameters, message)
}
func (e *Engine) RunReceiverHandlers(wsCtx wsContext.WSContext, message *Message) {
	e.runHandlers(wsCtx, e.ReceiveHandlers, &e.ReceiveParameters, message)
}
