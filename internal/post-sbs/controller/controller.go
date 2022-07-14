package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sbasestarter/post-sbs/internal/config"
	"github.com/sbasestarter/post/pkg/post"
	postpb "github.com/sbasestarter/proto-repo/gen/protorepo-post-go"
	postsbspb "github.com/sbasestarter/proto-repo/gen/protorepo-post-sbs-go"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/helper"
	"github.com/sgostarter/librediscovery"
	"github.com/sgostarter/libservicetoolset/clienttoolset"
	"google.golang.org/grpc"
)

const (
	gRPCSchema = "grpclb"

	serverNamePostKey = "post"
)

type Controller struct {
	cfg      *config.Config
	postConn *grpc.ClientConn
	postCli  postpb.PostServiceClient
	logger   l.WrapperWithContext
}

func NewController(ctx context.Context, cfg *config.Config, redisCli *redis.Client, logger l.Wrapper) *Controller {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	lc := logger.GetWrapperWithContext()

	getter, err := librediscovery.NewGetter(ctx, logger, redisCli,
		"", 5*time.Minute, time.Minute)
	if err != nil {
		lc.Fatalf(ctx, "new discovery getter failed: %v", err)

		return nil
	}

	err = clienttoolset.RegisterSchemas(ctx, &clienttoolset.RegisterSchemasConfig{
		Getter:  getter,
		Schemas: []string{gRPCSchema},
	}, logger)
	if err != nil {
		lc.Fatalf(ctx, "register schema failed: %v", err)

		return nil
	}

	postServerName, ok := cfg.DiscoveryServerNames[serverNamePostKey]
	if !ok || postServerName == "" {
		lc.Fatal(ctx, "no post server name config")

		return nil
	}

	postConn, err := clienttoolset.DialGRpcServerByName(gRPCSchema, postServerName, &cfg.GRpcClientConfigTpl, nil)
	if err != nil {
		lc.Fatalf(ctx, "dial %v failed: %v", postServerName, err)

		return nil
	}

	return &Controller{
		cfg:      cfg,
		postConn: postConn,
		postCli:  postpb.NewPostServiceClient(postConn),
		logger:   lc.WithFields(l.StringField(l.ClsKey, "Controller")),
	}
}

func (c *Controller) PostCode(ctx context.Context, protocol postsbspb.PostProtocolType,
	purpose postsbspb.PostPurposeType, to, code string, expiredTimestamp int64) error {
	switch protocol {
	case postsbspb.PostProtocolType_PostProtocolMail:
		return c.postEmail(ctx, purpose, to, code, expiredTimestamp)
	case postsbspb.PostProtocolType_PostProtocolSMS:
		return c.postSMS(ctx, purpose, to, code, expiredTimestamp)
	}

	c.logger.Errorf(ctx, "unknown protocol %v", protocol)

	return fmt.Errorf("未知的协议: %v", protocol)
}

func (c *Controller) post(ctx context.Context, req *postpb.SendTemplateRequest) error {
	var resp *postpb.SendTemplateResponse

	var err error

	helper.DoWithTimeout(ctx, 10*time.Second, func(ctx context.Context) {
		resp, err = c.postCli.SendTemplate(ctx, req)
	})

	if err != nil {
		c.logger.Errorf(ctx, "post send template failed: %v", err)

		return err
	}

	if resp == nil || resp.Status == nil {
		err = errors.New("post send template return none")
		c.logger.Error(ctx, err)

		return err
	}

	if resp.Status.Status != postpb.PostStatus_PS_SUCCESS {
		err = fmt.Errorf("post send template failed: %v", resp.Status.Msg)
		c.logger.Error(ctx, err)

		return err
	}

	return nil
}

func (c *Controller) postSMS(ctx context.Context, purpose postsbspb.PostPurposeType,
	to, code string, expiredTimestamp int64) error {
	var templateID string

	switch purpose {
	case postsbspb.PostPurposeType_PostPurposeRegister:
		templateID = "389596"
	case postsbspb.PostPurposeType_PostPurposeLogin:
		templateID = "860253"
	case postsbspb.PostPurposeType_PostPurposeResetPassword:
		templateID = "557914"
	default:
		c.logger.Errorf(ctx, "unknown purpose %v", purpose)

		return fmt.Errorf("未知的: %v", purpose)
	}

	req := &postpb.SendTemplateRequest{
		ProtocolType: post.ProtocolTypeSMS,
		To:           []string{to},
		TemplateId:   templateID,
		Vars: []string{
			code,
			fmt.Sprintf("%v", int(time.Until(time.Unix(expiredTimestamp, 0)).Minutes())),
		},
	}

	return c.post(ctx, req)
}

func (c *Controller) postEmail(ctx context.Context, purpose postsbspb.PostPurposeType,
	to, code string, expiredTimestamp int64) error {
	var subject string

	switch purpose {
	case postsbspb.PostPurposeType_PostPurposeRegister:
		subject = "羊米注册验证码"
	case postsbspb.PostPurposeType_PostPurposeLogin:
		subject = "羊米登录验证码"
	case postsbspb.PostPurposeType_PostPurposeResetPassword:
		subject = "羊米重置密码验证码"
	default:
		c.logger.Errorf(ctx, "unknown purpose %v", purpose)

		return fmt.Errorf("未知的: %v", purpose)
	}

	req := &postpb.SendTemplateRequest{
		ProtocolType: post.ProtocolTypeEmail,
		To:           []string{to},
		TemplateId:   "0",
		Vars: []string{
			subject,
			fmt.Sprintf("您的验证码为 %v, 将在 %v 过期", code, time.Unix(expiredTimestamp, 0)),
			c.cfg.Signer,
		},
	}

	return c.post(ctx, req)
}
