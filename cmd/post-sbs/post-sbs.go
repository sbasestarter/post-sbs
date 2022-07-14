package main

import (
	"context"
	"time"

	"github.com/sbasestarter/post-sbs/internal/config"
	"github.com/sbasestarter/post-sbs/internal/post-sbs/server"
	postsbspb "github.com/sbasestarter/proto-repo/gen/protorepo-post-sbs-go"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libconfig"
	"github.com/sgostarter/liblogrus"
	"github.com/sgostarter/librediscovery"
	"github.com/sgostarter/libservicetoolset/dbtoolset"
	"github.com/sgostarter/libservicetoolset/servicetoolset"
	"google.golang.org/grpc"
)

func main() {
	logger := l.NewWrapper(liblogrus.NewLogrus())
	logger.GetLogger().SetLevel(l.LevelDebug)

	var cfg config.Config

	_, err := libconfig.Load("config.yaml", &cfg)
	if err != nil {
		logger.Fatalf("load config failed: %v", err)

		return
	}

	dbToolset := dbtoolset.NewToolset(&cfg.DbConfig, logger)

	ctx := context.Background()

	cfg.GRpcServerConfig.DiscoveryExConfig.Setter, err = librediscovery.NewSetter(ctx, logger, dbToolset.GetRedis(),
		"", time.Minute)
	if err != nil {
		logger.Fatalf("create rediscovery setter failed: %v", err)

		return
	}

	serviceToolset := servicetoolset.NewServerToolset(context.Background(), logger)
	_ = serviceToolset.CreateGRpcServer(&cfg.GRpcServerConfig, nil, func(s *grpc.Server) error {
		postsbspb.RegisterPostSBSServiceServer(s, server.NewServer(context.Background(), &cfg, dbToolset.GetRedis(), logger))

		return nil
	})

	serviceToolset.Wait()
}
