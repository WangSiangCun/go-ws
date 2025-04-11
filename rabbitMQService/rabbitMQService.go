package rabbitMQService

import (
	"fmt"
	"github.com/streadway/amqp"
	"log"
)

func InitRabbitMQ(MQUrl string) *amqp.Connection {
	conn, err := amqp.Dial(MQUrl)
	failOnError(err, "Failed to connect to RabbitMQ")
	return conn
}

//我们还需要一个辅助函数来检查每个 amqp 调用的返回值：

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

// Destroy 断开channel 和 connection
func Destroy(channel *amqp.Channel, connection *amqp.Connection) {
	if err := channel.Close(); err != nil {
		return
	}
	if err := connection.Close(); err != nil {
		return
	}
}

// PublishRouting 路由模式发送消息
func PublishRouting(conn *amqp.Connection, exchange, key string, message []byte) {
	//通道将打开唯一的并发服务器通道，以处理大量 AMQP 消息。此接收器上的方法的任何错误都将使接收器无效，并且应打开一个新通道
	channel, err := conn.Channel()
	failOnError(err, "Failed to Get Channel for RabbitMQ")

	//1.尝试创建交换机
	err = channel.ExchangeDeclare(
		exchange,
		//要改成direct
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)

	failOnError(err, "Failed to declare an exchange:"+exchange)

	//2.发送消息
	err = channel.Publish(
		exchange,
		//要设置
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		})
}

// ReceiveRouting 路由模式接受消息
func ReceiveRouting(conn *amqp.Connection, exchange, key string) (messages <-chan amqp.Delivery, err error) {
	channel, err := conn.Channel()
	//1.试探性创建交换机
	err = channel.ExchangeDeclare(
		exchange,
		//交换机类型
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare an exchange")
	//2.试探性创建队列，这里注意队列名称不要写
	q, err := channel.QueueDeclare(
		"MessageToClient", //随机生产队列名称
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	//绑定队列到 exchange 中
	err = channel.QueueBind(
		q.Name,
		//需要绑定key
		key,
		exchange,
		false,
		nil)

	//4.消费代码
	//4.1接收队列消息
	message, err := channel.Consume(
		//队列名称
		q.Name,
		//用来区分多个消费者
		"client",
		//是否自动应答 意思就是收到一个消息已经被消费者消费完了是否主动告诉rabbitmq服务器我已经消费完了你可以去删除这个消息啦 默认是true
		false,
		//是否具有排他性
		false,
		//如果设置为true表示不能将同一个connection中发送的消息传递给同个connection中的消费者
		false,
		//队列消费是否阻塞 fase表示是阻塞 true表示是不阻塞
		false,
		nil,
	)
	if err != nil {
		fmt.Println(err)
	}

	//for m := range message {
	//	fmt.Println(string(m.Body))
	//}

	return message, nil

}

// PublishWorking 工作模式发送消息
func PublishWorking(conn *amqp.Connection, queueName string, message []byte) {
	channel, err := conn.Channel()
	q, err := channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = channel.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(message),
		})
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %s", message)

}

// ReceiveWorking 工作模式接收消息
func ReceiveWorking(conn *amqp.Connection, queueName string) (messages <-chan amqp.Delivery, err error) {
	channel, err := conn.Channel()
	q, err := channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	failOnError(err, "Failed to declare a queue")

	err = channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	messages, err = channel.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")
	return
}

/*
这段代码是使用 AMQP 协议连接到 RabbitMQ 服务器，并声明一个队列，设置 QoS 和注册一个消费者。

下面是各个字段的含义：
- `queueName`：队列的名称。
- `durable`：队列是否持久化。如果设置为 true，则队列将在服务器重启后继续存在。
- `deleteWhenUnused`：当没有消费者连接到队列时，是否自动删除队列。
- `exclusive`：是否将队列标记为独占队列。如果设置为 true，则只有当前连接的消费者可以使用该队列。
- `noWait`：是否等待服务器响应。如果设置为 true，则不会等待服务器响应，而是立即返回。
- `arguments`：队列的其他属性。可以设置队列的最大长度、过期时间等。
- `prefetchCount`：每个消费者在确认之前可以接收的最大消息数。这个值可以控制消费者的负载均衡。
- `prefetchSize`：每个消费者在确认之前可以接收的最大消息大小。通常设置为 0，表示不限制消息大小。
- `global`：是否将 QoS 设置为全局。如果设置为 true，则将 QoS 应用于所有消费者，而不是单个消费者。
- `queue`：要消费的队列的名称。
- `consumer`：消费者的名称。如果设置为空字符串，则服务器将为消费者生成一个唯一的名称。
- `autoAck`：是否自动确认消息。如果设置为 true，则消费者在接收到消息后立即确认消息。如果设置为 false，则消费者必须手动确认消息。
- `noLocal`：是否禁止消费者接收自己发布的消息。
- `noWait`：是否等待服务器响应。如果设置为 true，则不会等待服务器响应，而是立即返回。
- `args`：消费者的其他属性。可以设置消费者的优先级、消息过滤等。
*/
