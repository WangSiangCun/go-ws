package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/WangSiangCun/go-ws/config"
	"github.com/WangSiangCun/go-ws/core/conf"
	"github.com/WangSiangCun/go-ws/engine"
	"github.com/WangSiangCun/go-ws/hub"
	"github.com/WangSiangCun/go-ws/wsContext"
	"net/http"
)

var configFile = flag.String("f", "etc/webSocketService.yaml", "the config file")

func main() {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c)
	e := engine.NewEngine(&c)
	// ... existing code ...
	//  接收消息后插件
	//e.OpenJWT(config.JWT{c.JWT.AccessSecret, c.JWT.AccessExpire})
	e.SetReceiverHandlers([]engine.HandlersFunc{ReceiveHandler})
	receiverParameters := [][]any{[]any{&engine.Message{Type: 1}}}
	e.SetReceiverParameters(receiverParameters)
	//	e.RunHandlers(wsContext.NewContext(context.Background(), &c), e.ReceiveHandlers, e.ReceiveParameters, &engine.Message{})

	//  发送消息前插件
	sendParameters := [][]any{[]any{1}}
	e.SetSendHandlers([]engine.HandlersFunc{SendHandler})
	e.SetSendParameters(sendParameters)

	// ... existing code ...
	hub := hub.NewHub()
	go hub.Run(wsContext.NewContext(context.Background(), &c), e)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWs(w, r, e)
	})
	err := http.ListenAndServe(c.WSPort, nil)
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
		return
	}

}

const (
	WebSocketMessageTypeChat  = iota // websocket message type chat
	WebSocketMessageTypeMatch        // websocket message type match
)

func SendHandler(e *engine.Engine, wsCtx wsContext.WSContext, message *engine.Message, opts ...any) *engine.Message {

	if message.Type == WebSocketMessageTypeChat {
		// 聊天类型消息
		fmt.Println("chat message")
	} else if message.Type == WebSocketMessageTypeMatch {
		// 截断，只和server

		// 匹配类型消息

		fmt.Println("chat message")
		e.IsServerHandlerModel = true

		// 执行您的逻辑
		wsCtx.EtcdClient.Put(wsCtx.Context, message.SourceId, "test")
		fmt.Println("serverHandler, this message is server handler")

		/*
		  这里可以写一些逻辑，比如：
		  1. 过滤掉一些消息
		  2. webSocket的server逻辑，比如游戏服务器相关逻辑等等之类
		  3. 可以把消息转发转存给其他服务  如mysql，es，总之，一切看自己的需求
		*/
	}
	return message
}

func ReceiveHandler(e *engine.Engine, wsCtx wsContext.WSContext, message *engine.Message, opts ...any) *engine.Message {
	// 目标用户接收到消息前的逻辑
	/* 1.可以完成加解密
	   2.可以完成一些业务逻辑
	*/

	if len(opts) != 0 {
		if a, ok := opts[0].(*engine.Message); ok {
			fmt.Println("message type:", a.Type)
		}

	}
	fmt.Println("receive a message")

	return message
}
