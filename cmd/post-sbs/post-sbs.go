package main

import (
	"context"
	"time"

	"github.com/jiuzhou-zhao/go-fundamental/dbtoolset"
	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/jiuzhou-zhao/go-fundamental/servicetoolset"
	"github.com/jiuzhou-zhao/go-fundamental/tracing"
	"github.com/sbasestarter/post-sbs/internal/config"
	"github.com/sbasestarter/post-sbs/internal/post-sbs/server"
	"github.com/sbasestarter/proto-repo/gen/protorepo-post-sbs-go"
	"github.com/sgostarter/libconfig"
	"github.com/sgostarter/liblog"
	"github.com/sgostarter/librediscovery"
	"google.golang.org/grpc"
)

func main() {
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

	ctx := context.Background()
	dbToolset, err := dbtoolset.NewDBToolset(ctx, &cfg.DBConfig, loggerChain)
	if err != nil {
		loge.Fatalf(context.Background(), "db toolset create failed: %v", err)
		return
	}
	cfg.GRpcServerConfig.DiscoveryExConfig.Setter, err = librediscovery.NewSetter(ctx, loggerChain, dbToolset.GetRedis(),
		"", time.Minute)
	if err != nil {
		loge.Fatalf(context.Background(), "create rediscovery setter failed: %v", err)
		return
	}

	serviceToolset := servicetoolset.NewServerToolset(context.Background(), loggerChain)
	_ = serviceToolset.CreateGRpcServer(&cfg.GRpcServerConfig, nil, func(s *grpc.Server) {
		postsbspb.RegisterPostSBSServiceServer(s, server.NewServer(context.Background(), &cfg, dbToolset))
	})
	serviceToolset.Wait()
}
