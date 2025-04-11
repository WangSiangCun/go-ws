# go-ws

go-ws 是一个基于Go语言实现的分布式WebSocket服务器框架，它提供了完整的WebSocket连接管理和消息处理机制，支持多服务器部署和分布式消息处理。

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
go get github.com/your-username/go-ws
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
    "flag"
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
    engine := engine.NewEngine(&c)

    hub := hub.NewHub()
    go hub.Run(wsContext.NewContext(context.Background(), &c), engine)

    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        hub.ServeWs(w, r, engine)
    })
    http.ListenAndServe(c.Port, nil)
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
type MessageHandler interface {
    Handle(message *engine.Message) *engine.Message
}
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

