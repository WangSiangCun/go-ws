package wsContext

import (
	"context"
	"github.com/streadway/amqp"
	"go-ws/config"
	"go-ws/etcdService"
	"go-ws/rabbitMQService"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type WSContext struct {
	Context            context.Context
	EtcdClient         *clientv3.Client
	RabbitMQConnection *amqp.Connection
}

func NewContext(c context.Context, config *config.Config) WSContext {
	etcdClient := etcdService.MustInitEtcd(config.Etcd.Hosts)
	rabbitMQConnect := rabbitMQService.InitRabbitMQ(config.RabbitMQ.MQUrl)
	return WSContext{
		EtcdClient:         etcdClient,
		RabbitMQConnection: rabbitMQConnect,
		Context:            c,
	}
}
