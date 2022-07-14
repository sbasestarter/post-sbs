package server

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/sbasestarter/post-sbs/internal/config"
	"github.com/sbasestarter/post-sbs/internal/post-sbs/controller"
	postsbspb "github.com/sbasestarter/proto-repo/gen/protorepo-postsbs-go"
	sharepb "github.com/sbasestarter/proto-repo/gen/protorepo-share-go"
	"github.com/sgostarter/i/l"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	controller *controller.Controller
}

func NewServer(ctx context.Context, cfg *config.Config, redisCli *redis.Client, logger l.Wrapper) *Server {
	return &Server{
		controller: controller.NewController(ctx, cfg, redisCli, logger),
	}
}

func (s *Server) PostCode(ctx context.Context, req *postsbspb.PostCodeRequest) (*sharepb.Empty, error) {
	err := s.controller.PostCode(ctx, req.ProtocolType, req.PurposeType, req.To, req.Code, req.ExpiredTimestamp)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &sharepb.Empty{}, nil
}
