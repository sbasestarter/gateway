package main

import (
	"context"

	"github.com/jiuzhou-zhao/go-fundamental/dbtoolset"
	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/jiuzhou-zhao/go-fundamental/servicetoolset"
	"github.com/jiuzhou-zhao/go-fundamental/tracing"
	"github.com/sbasestarter/gateway/internal/config"
	"github.com/sbasestarter/gateway/internal/gateway"
	"github.com/sgostarter/libconfig"
	"github.com/sgostarter/liblog"
	"github.com/trusch/grpc-proxy/proxy"
	"google.golang.org/grpc"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := liblog.NewZapLogger()
	if err != nil {
		panic(err)
	}
	loggerChain := loge.NewLoggerChain()
	loggerChain.AppendLogger(tracing.NewTracingLogger())
	loggerChain.AppendLogger(logger)
	loge.SetGlobalLogger(loge.NewLogger(loggerChain))

	var cfg config.Config
	_, err = libconfig.Load("config", &cfg)
	if err != nil {
		loge.Fatalf(context.Background(), "load config failed: %v", err)
		return
	}

	dbToolset, err := dbtoolset.NewDBToolset(ctx, &cfg.DBConfig, loggerChain)
	if err != nil {
		loge.Fatalf(context.Background(), "db toolset create failed: %v", err)
		return
	}

	gw := gateway.NewGateway(context.Background(), &cfg, dbToolset)
	opts := []grpc.ServerOption{
		grpc.UnknownServiceHandler(proxy.TransparentHandler(gw.GetGRpcDirector())),
	}

	serviceToolset := servicetoolset.NewServerToolset(context.Background(), logger)
	_ = serviceToolset.CreateGRpcServer(&cfg.GRpcServerConfig, opts, func(server *grpc.Server) {

	})
	serviceToolset.Wait()
}
