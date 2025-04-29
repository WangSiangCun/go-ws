# go-ws

go-ws 是一个基于Go语言实现的分布式WebSocket服务器框架，它提供了完整的WebSocket连接管理和消息处理机制，支持多服务器部署和分布式消息处理。
![image](https://github.com/user-attachments/assets/9cd5b1c3-bbc2-4ed3-a756-936528bce840)


## 特性

- 🚀 **分布式支持**：基于etcd实现服务发现，支持多服务器部署
- 🔄 **消息队列**：集成RabbitMQ，实现可靠的消息传递
- 🔒 **安全认证**：支持JWT认证，可配置的鉴权机制
- 🧩 **插件化**：支持自定义消息处理器，灵活扩展
- 📦 **简单易用**：通过简单的配置即可快速搭建WebSocket服务
- 🔌 **高可用**：支持服务发现和负载均衡

## 快速开始

### 安装

```bash
go get github.com/WangSiangCun/go-ws
```

### 配置

创建配置文件 `etc/webSocketService.yaml`：

```yaml
port: ":8080"
host: "localhost"
isOpenJWT: true
jwt:
  accessSecret: "your-secret-key"
etcd:
  endpoints:
    - "localhost:2379"
rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
```

### 使用示例

```go
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

const (
	WebSocketMessageTypeChat  = iota // websocket message type chat
	WebSocketMessageTypeMatch        // websocket message type match
)

func SendHandler(e *engine.Engine, wsCtx wsContext.WSContext, message *engine.Message) *engine.Message {

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

func ReceiveHandler(e *engine.Engine, wsCtx wsContext.WSContext, message *engine.Message) *engine.Message {
	// 目标用户接收到消息前的逻辑
	/* 1.可以完成加解密
	   2.可以完成一些业务逻辑
	*/
	fmt.Println("receive a message")

	return message
}

```

## 架构设计

### 核心组件

- **Hub**: 管理WebSocket连接和消息分发
- **Engine**: 提供消息处理核心逻辑
- **Context**: 管理全局上下文和服务发现
- **Client**: 处理单个WebSocket连接

### 消息流程

1. 客户端通过WebSocket连接到服务器
2. 消息通过Hub进行分发
3. Engine处理消息并应用插件
4. 通过RabbitMQ在服务器间传递消息
5. 目标服务器接收并转发消息到对应客户端

## 配置说明

### 主要配置项

- `port`: WebSocket服务端口
- `host`: 服务器主机地址
- `isOpenJWT`: 是否启用JWT认证
- `jwt.accessSecret`: JWT密钥
- `etcd.endpoints`: etcd服务地址
- `rabbitmq.url`: RabbitMQ连接地址

## 插件开发

go-ws支持自定义消息处理器，实现`MessageHandler`接口即可：

```go
package main

import (
	"context"
	"flag"
	"fmt"
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
		fmt.Println("serverHandler")

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



```

res:
```shell
SendChannel 0.0.0.0:4444 &{aa [1910314082062307328] 1910306341868539904}
2025/04/12 00:02:39  [x] Sent {"message":"aa","target_ids":["1910314082062307328"],"source_id":"1910306341868539904"}
{"message":"aa","target_ids":["1910314082062307328"],"source_id":"1910306341868539904"}
ReadChannel 0.0.0.0:4444 &{aa [1910314082062307328] 1910306341868539904}
serverHandler, this message is server handler
SendChannel 0.0.0.0:4444 &{aa [server] 1910314082062307328}
On 1910314082062307328
On 1910306341868539904
On 1910314082062307328

```



## 贡献指南

欢迎提交Issue和Pull Request！

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 联系方式

- 邮箱：[wangxiangkunacmer@gmail.com]
- 持续更新中，期待你的关注和支持！

