package etcdService

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/client/v3"
	"log"
	"time"
)

func MustInitEtcd(hosts []string) (cli *clientv3.Client) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   hosts,
		DialTimeout: 10 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	return
}
func SetKeepLiveLease(cli *clientv3.Client, seconds int64, key string, value string) {
	// 创建一个5秒的租约
	resp, err := cli.Grant(context.TODO(), seconds)
	if err != nil {
		log.Fatal(err)
	}

	//key = "/chat/" + client.chatId
	//val := svc.Config.Host + ":" + strconv.Itoa(svc.Config.Port)
	_, err = cli.Put(context.TODO(), key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
	}
	// the key 'foo' will be kept forever
	_, err = cli.KeepAlive(context.TODO(), resp.ID)
	if err != nil {
		log.Fatal(err)
	}
}
func SetLease(cli *clientv3.Client, seconds int64, key string, value string) clientv3.LeaseID {
	// 创建一个5秒的租约
	resp, err := cli.Grant(context.TODO(), seconds)
	if err != nil {
		log.Fatal(err)
	}
	_, err = cli.Put(context.TODO(), key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
	}
	return resp.ID
}
func CancelLease(cli *clientv3.Client, leaseId clientv3.LeaseID) {
	_, err := cli.Revoke(context.TODO(), leaseId)
	if err != nil {
		fmt.Printf("revoke failed, err:%v\n", err)
	}
}
func Get(cli *clientv3.Client, key string) *clientv3.GetResponse {
	// get
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	resp, err := cli.Get(ctx, key)
	cancel() //在这段代码中，`cancel()` 的作用是取消 `ctx` 上下文的操作。在这里，`ctx` 是一个带有超时时间的上下文，它会在超时时间到达时自动取消。但是，如果操作在超时时间之前完成，我们需要手动取消上下文以避免资源泄漏。因此，我们使用 `cancel()` 函数来取消上下文。这将确保在操作完成后，上下文及其相关资源被正确释放。
	if err != nil {
		fmt.Printf("get from etcd failed, err:%v\n", err)
		return nil
	}
	return resp
}
