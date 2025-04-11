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
	engine := engine.NewEngine(&c)

	hub := hub.NewHub()
	go hub.Run(wsContext.NewContext(context.Background(), &c), engine)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWs(w, r, engine)
	})
	err := http.ListenAndServe(c.Port, nil)
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
		return
	}

}
