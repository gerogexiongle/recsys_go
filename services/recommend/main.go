package main

import (
	"flag"
	"fmt"

	"log"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"

	"recsys_go/services/recommend/internal/config"
	"recsys_go/services/recommend/internal/handler"
	"recsys_go/services/recommend/internal/svc"
)

var configFile = flag.String("f", "etc/recommend-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	config.ApplyEnvOverrides(&c)

	srv := rest.MustNewServer(c.RestConf)
	defer srv.Stop()

	ctx, err := svc.NewServiceContext(c, *configFile)
	if err != nil {
		log.Fatal(err)
	}
	handler.RegisterHandlers(srv, ctx)

	fmt.Printf("recommend-api listening on %s:%d\n", c.Host, c.Port)
	srv.Start()
}
