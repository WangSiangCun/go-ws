package hub

import (
	"encoding/json"
	"fmt"
	"github.com/WangSiangCun/go-ws/core/jwtHelper"
	"github.com/WangSiangCun/go-ws/engine"
	"github.com/WangSiangCun/go-ws/etcdService"
	"github.com/WangSiangCun/go-ws/rabbitMQService"
	"github.com/WangSiangCun/go-ws/wsContext"
	"github.com/golang-jwt/jwt/v4"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"net/http"
	"strings"
	"time"
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
func (h *Hub) RemoveClient(clientId string) {
	h.Clients[clientId] = nil
}
func (h *Hub) RegisterClient(wsContext wsContext.WSContext, hub *Hub, e *engine.Engine, clientId string, c *Client) {
	hub.Clients[clientId] = c
	//注册用户在etcd上
	ticker := time.Tick(10 * time.Second)
	var leaseId clientv3.LeaseID
	//立刻设置租约，不然要等五秒
	leaseId = etcdService.SetLease(wsContext.EtcdClient, e.Config.PongTime, clientId, e.Config.Host+e.Config.WSPort)
	fmt.Println("register:" + clientId)
	for {
		select {
		case <-ticker:
			// 每隔 10 秒执行一次该操作
			fmt.Println("On", c.Id)
			leaseId = etcdService.SetLease(wsContext.EtcdClient, e.Config.PongTime, clientId, e.Config.Host+e.Config.WSPort)
		case <-c.ToOffline:
			//退出直接取消租约，设置租约时间只是保障
			etcdService.CancelLease(wsContext.EtcdClient, leaseId)
			return
		}
	}

}
func (h *Hub) SendMessage(ws wsContext.WSContext, message *engine.Message, e *engine.Engine) {
	// 插件
	e.RunSendHandlers(ws, message)
	if e.IsServerHandlerModel {
		// 如果是Server handler 模式，就不继续执行了
		return
	}

	fmt.Println("SendChannel", e.Config.Host+e.Config.WSPort, message)
	serverToMessage := map[string]*engine.Message{}
	if message == nil {
		log.Println("消息为空")
		return
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
}
func (h *Hub) ReceiveMessage(ws wsContext.WSContext, message *engine.Message, e *engine.Engine) {
	// 插件
	e.RunReceiverHandlers(ws, message)

	if message == nil {
		log.Println("消息为空")
		return
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
func (h *Hub) Run(ws wsContext.WSContext, e *engine.Engine) {
	for {
		select {
		case client := <-h.Register:
			h.RegisterClient(ws, h, e, client.Id, client)
		case client := <-h.Unregister:
			h.RemoveClient(client.Id)
		case message, ok := <-h.SendChannel:
			if !ok {
				log.Println("SendChannel通道关闭")
			}
			fmt.Println("SendChannel", e.Config.Host+e.Config.WSPort, message)
			go h.SendMessage(ws, message, e)
		case message, ok := <-h.ReadChannel:
			if !ok {
				log.Println("ReadChannel通道关闭")
			}
			fmt.Println("ReadChannel", e.Config.Host+e.Config.WSPort, message)
			go h.ReceiveMessage(ws, message, e)
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
	// myMap 包含 key 值，val 为对应的 value 值
	if h.Clients[clientId] != nil {
		// 存在 key
		h.Clients[clientId].Close()
	}

	client := NewClient(h, conn, clientId)
	client.Hub.Register <- client
	go client.readPump(wsContext)
	go client.writePump(wsContext)
	go client.ReceiveMessage(wsContext, e)
	go client.SendMessage(wsContext, e)
}
