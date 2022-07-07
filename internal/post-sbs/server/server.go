package server

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/sbasestarter/post-sbs/internal/config"
	"github.com/sbasestarter/post-sbs/internal/post-sbs/controller"
	"github.com/sbasestarter/proto-repo/gen/protorepo-post-sbs-go"
	"github.com/sgostarter/i/l"
)

type Server struct {
	controller *controller.Controller
}

func NewServer(ctx context.Context, cfg *config.Config, redisCli *redis.Client, logger l.Wrapper) *Server {
	return &Server{
		controller: controller.NewController(ctx, cfg, redisCli, logger),
	}
}

func (s *Server) PostCode(ctx context.Context, req *postsbspb.PostCodeRequest) (*postsbspb.PostCodeResponse, error) {
	err := s.controller.PostCode(ctx, req.ProtocolType, req.PurposeType, req.To, req.Code, req.ExpiredTimestamp)
	status := postsbspb.PostSBSStatus_PS_SBS_SUCCESS
	msg := ""
	if err != nil {
		status = postsbspb.PostSBSStatus_PS_SBS__FAILED
		msg = err.Error()
	}
	return &postsbspb.PostCodeResponse{
		Status: &postsbspb.ServerStatus{
			Status: status,
			Msg:    msg,
		},
	}, nil
}
