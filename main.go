package main

import (
	"context"
	"flag"
	"fmt"
	"go-ws/config"
	"go-ws/core/conf"
	"go-ws/engine"
	"go-ws/hub"
	"go-ws/wsContext"
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
	e.SetReadHandlers([]engine.HandlersFunc{ReceiveHandler})

	//  发送消息前插件
	e.SetSendHandlers([]engine.HandlersFunc{SendHandler})
	// ... existing code ...
	hub := hub.NewHub()
	go hub.Run(wsContext.NewContext(context.Background(), &c), e)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWs(w, r, e)
	})
	err := http.ListenAndServe(c.Port, nil)
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
		return
	}

}

func SendHandler(e *engine.Engine, wsCtx wsContext.WSContext, message *engine.Message) *engine.Message {
	// 截断，只和server通信的例子
	if len(message.TargetIds) == 1 && message.TargetIds[0] == "server" {
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

func ReceiveHandler(e *engine.Engine, wsCtx wsContext.WSContext, message *engine.Message) *engine.Message {
	// 目标用户接收到消息前的逻辑
	/* 1.可以完成加解密
	   2.可以完成一些业务逻辑
	*/
	fmt.Println("receive a message")

	return message
}
