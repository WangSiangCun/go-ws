package hub

import (
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"go-ws/core/jwtHelper"
	"go-ws/engine"
	"go-ws/rabbitMQService"
	"go-ws/wsContext"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"net/http"
	"strings"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	Clients map[string]*Client

	// Inbound messages from the clients.
	SendChannel chan *engine.Message
	ReadChannel chan *engine.Message
	// Register requests from the clients.
	Register chan *Client

	// Unregister requests from clients.
	Unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		SendChannel: make(chan *engine.Message),
		ReadChannel: make(chan *engine.Message),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		Clients:     make(map[string]*Client),
	}
}
func (h *Hub) Run(ws wsContext.WSContext, e *engine.Engine) {
	for {
		select {
		case client := <-h.Register:
			go client.Register(h, ws, client.Id, e)
		case client := <-h.Unregister:
			fmt.Println("断开连接", client.Id)
			if _, ok := h.Clients[client.Id]; ok {
				//取消注册
				//删除指定client
				delete(h.Clients, client.Id)
				//关闭channel
				close(client.WriteChannel)
			}
		case message, ok := <-h.SendChannel:
			if !ok {
				log.Println("SendChannel通道关闭")
			}
			// 插件
			for _, msgHandler := range e.SendHandlers {
				message = msgHandler(message)
			}
			fmt.Println("SendChannel", e.Config.Host+e.Config.Port, message)
			serverToMessage := map[string]*engine.Message{}
			if message == nil {
				log.Println("消息为空")
				break
			}
			// 按服务器：IP分组 重新组装message
			for _, targetId := range message.TargetIds {
				get, err := ws.EtcdClient.Get(ws.Context, targetId, clientv3.WithPrefix())
				if err != nil {
					return
				}
				if len(get.Kvs) == 0 {
					continue
				}
				serverIPANDHost := string(get.Kvs[0].Value)
				if serverToMessage[serverIPANDHost] == nil {
					serverToMessage[serverIPANDHost] = &engine.Message{
						Message:   message.Message,
						TargetIds: []string{targetId},
						SourceId:  message.SourceId,
					}
				} else {
					serverToMessage[serverIPANDHost].TargetIds = append(serverToMessage[serverIPANDHost].TargetIds, targetId)
				}
			}
			// 发送消息
			for serverIPANDHost, toMessage := range serverToMessage {
				messageByte, err := json.Marshal(toMessage)
				if err != nil {
					log.Printf("error: %v", err)
				}
				rabbitMQService.PublishWorking(ws.RabbitMQConnection, serverIPANDHost, messageByte)
			}
		case message, ok := <-h.ReadChannel:
			if !ok {
				log.Println("ReadChannel通道关闭")
			}
			fmt.Println("ReadChannel", e.Config.Host+e.Config.Port, message)

			// 插件
			for _, msgHandler := range e.SendHandlers {
				message = msgHandler(message)
			}
			if message == nil {
				log.Println("消息为空")
				continue
			}
			// 发送给对应的客户端
			for _, targetId := range message.TargetIds {
				if client, ok := h.Clients[targetId]; ok {
					jsonMessage, err := json.Marshal(message)
					if err != nil {
						log.Printf("error: %v", err)
					}
					client.WriteChannel <- jsonMessage
				}
			}

		}

	}
}
func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request, e *engine.Engine) {
	wsContext := wsContext.NewContext(r.Context(), e.Config)
	token, found := strings.CutPrefix(r.URL.RawQuery, "token=")
	claims := &jwt.MapClaims{}
	clientId := ""
	ok := false
	//鉴权，判断是是否登录
	if e.IsOpenJWT {
		isValid := false
		isValid, claims = jwtHelper.IsValid(e.Config.JWT.AccessSecret, token)
		isValid = true
		if !isValid {
			_, err := w.Write([]byte("鉴权失败"))
			fmt.Println(err)
			return
		}
		clientId, ok = (*claims)["client_id"].(string)
		if !ok {
			_, err := w.Write([]byte("鉴权失败"))
			fmt.Println(err)
			return
		}
	} else {
		clientId, found = strings.CutPrefix(r.URL.RawQuery, "client_id=")
		if found == false {
			return
		}
	}
	//升级协议
	conn, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	//chatId := r.Header["Chatid"][0] //r.Header只有首字母大写

	// myMap 包含 key 值，val 为对应的 value 值

	client := &Client{
		Hub:          h,
		Conn:         conn,
		WriteChannel: make(chan []byte, bufSize),
		Id:           clientId,
		ToOffline:    make(chan bool),
	}
	client.Hub.Register <- client
	go client.readPump(wsContext)
	go client.writePump(wsContext)
	go client.ReceiveMessage(wsContext, e)
}
