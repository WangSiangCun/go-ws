package hub

import (
	"encoding/json"
	"fmt"
	"github.com/WangSiangCun/go-ws/engine"
	"github.com/WangSiangCun/go-ws/etcdService"
	"github.com/WangSiangCun/go-ws/rabbitMQService"
	"github.com/WangSiangCun/go-ws/wsContext"
	"github.com/gorilla/websocket"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 2048

	// send buffer size
	bufSize = 2048
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upGrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	Hub *Hub
	// The websocket connection.
	Conn *websocket.Conn

	WriteChannel chan []byte

	// Buffered channel of outbound messages.
	Id string

	ToOffline chan bool

	LeaseId *clientv3.LeaseID

	SendHandlers    []func(msg *engine.Message) *engine.Message
	ReceiveHandlers []func(msg *engine.Message) *engine.Message
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
var unOnlineMutex sync.Mutex

func (c *Client) readPump(wsContext wsContext.WSContext) {
	defer func() {
		//断开链接默认会走这里
		unOnlineMutex.Lock() //加锁，避免不同步
		c.Hub.Unregister <- c
		c.ToOffline <- true
		unOnlineMutex.Unlock()
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, messageByte, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return
			}
			log.Printf("error: %v", err)
			//return
		}
		c.SendMessage(wsContext, messageByte)
	}
}
func (c *Client) writePump(wsContext wsContext.WSContext) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.WriteChannel:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.WriteChannel)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.WriteChannel)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}

}
func (c *Client) SendMessage(wsContext wsContext.WSContext, messageByte []byte) {
	message := &engine.Message{}
	err := json.Unmarshal(messageByte, message)
	if err != nil {
		log.Printf("error: %v", err)
		return
	}
	//  插件
	for _, msgHandler := range c.SendHandlers {
		message = msgHandler(message)
	}
	// 进入中转中心
	c.Hub.SendChannel <- message
}
func (c *Client) ReceiveMessage(wsContext wsContext.WSContext, e *engine.Engine) {
	working, err := rabbitMQService.ReceiveWorking(wsContext.RabbitMQConnection, e.Config.Host+e.Config.WSPort)
	if err != nil {
		log.Panicln("启动接收错误")
	}
	for mqMessage := range working {
		message := &engine.Message{}
		err = json.Unmarshal(mqMessage.Body, message)
		fmt.Println(string(mqMessage.Body))
		//  执行插件
		for _, msgHandler := range c.ReceiveHandlers {
			message = msgHandler(message)
		}
		//handler.ChatMessageHandler(ctx, message.Body)
		//必须，否则会导致无法正常接收
		err := mqMessage.Ack(false)
		if err != nil {
			fmt.Println(err)
		}
		c.Hub.ReadChannel <- message
	}

}
func (c *Client) Register(hub *Hub, wsContext wsContext.WSContext, clientId string, e *engine.Engine) {
	hub.Clients[c.Id] = c
	//注册用户在etcd上
	ticker := time.Tick(10 * time.Second)
	var leaseId clientv3.LeaseID
	//立刻设置租约，不然要等五秒
	leaseId = etcdService.SetLease(wsContext.EtcdClient, e.Config.PongTime, clientId, e.Config.Host+e.Config.WSPort)
	for {
		select {
		case <-ticker:
			// 每隔 10 秒执行一次该操作
			fmt.Println("On", c.Id)
			leaseId = etcdService.SetLease(wsContext.EtcdClient, e.Config.PongTime, clientId, e.Config.Host+e.Config.WSPort)
		case <-c.ToOffline:
			fmt.Println("exit")
			etcdService.CancelLease(wsContext.EtcdClient, leaseId)
			hub.Clients[c.Id] = nil
			//退出直接取消租约，设置租约时间只是保障
			return
		}
	}

}
