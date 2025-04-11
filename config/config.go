package config

type JWT struct {
	AccessSecret string
	AccessExpire int64
}
type Etcd struct {
	Hosts []string
}
type RabbitMQ struct {
	Hosts    []string
	MQUrl    string
	Exchange string
}
type Config struct {
	Etcd     Etcd
	RabbitMQ RabbitMQ
	Host     string
	Port     string
	PongTime int64
	JWT      JWT
}

func NewConfig(etcdHost []string, mqUrl string, host string, post string, pongTime int64) *Config {
	return &Config{Etcd{Hosts: etcdHost}, RabbitMQ{MQUrl: mqUrl}, host, post, pongTime, JWT{}}
}
