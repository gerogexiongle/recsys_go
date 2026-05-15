package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"

	"recsys_go/services/rank/internal/config"
	"recsys_go/services/rank/internal/handler"
	"recsys_go/services/rank/internal/svc"
)

var configFile = flag.String("f", "etc/rank-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	srv := rest.MustNewServer(c.RestConf)
	defer srv.Stop()

	ctx, err := svc.NewServiceContext(c, *configFile)
	if err != nil {
		log.Fatalf("%v", err)
	}
	handler.RegisterHandlers(srv, ctx)

	fmt.Printf("rank-api listening on %s:%d\n", c.Host, c.Port)
	srv.Start()
}
