package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiuzhou-zhao/go-fundamental/clienttoolset"
	"github.com/jiuzhou-zhao/go-fundamental/dbtoolset"
	"github.com/jiuzhou-zhao/go-fundamental/loge"
	"github.com/jiuzhou-zhao/go-fundamental/utils"
	"github.com/sbasestarter/post-sbs/internal/config"
	"github.com/sbasestarter/post/pkg/post"
	"github.com/sbasestarter/proto-repo/gen/protorepo-post-go"
	"github.com/sbasestarter/proto-repo/gen/protorepo-post-sbs-go"
	"github.com/sgostarter/librediscovery"
	"google.golang.org/grpc"
)

const (
	gRpcSchema = "grpclb"

	serverNamePostKey = "post"
)

type Controller struct {
	cfg      *config.Config
	postConn *grpc.ClientConn
	postCli  postpb.PostServiceClient
}

func NewController(ctx context.Context, cfg *config.Config, dbToolset *dbtoolset.DBToolset) *Controller {
	getter, err := librediscovery.NewGetter(ctx, loge.GetGlobalLogger().GetLogger(), dbToolset.GetRedis(),
		"", 5*time.Minute, time.Minute)
	if err != nil {
		loge.Fatalf(ctx, "new discovery getter failed: %v", err)
		return nil
	}

	err = clienttoolset.RegisterSchemas(ctx, &clienttoolset.RegisterSchemasConfig{
		Getter:  getter,
		Logger:  loge.GetGlobalLogger().GetLogger(),
		Schemas: []string{gRpcSchema},
	})
	if err != nil {
		loge.Fatalf(ctx, "register schema failed: %v", err)
		return nil
	}
	postServerName, ok := cfg.DiscoveryServerNames[serverNamePostKey]
	if !ok || postServerName == "" {
		loge.Fatal(ctx, "no post server name config")
		return nil
	}
	postConn, err := clienttoolset.DialGRpcServerByName(gRpcSchema, postServerName, &cfg.GRpcClientConfigTpl, nil)
	if err != nil {
		loge.Fatalf(ctx, "dial %v failed: %v", postServerName, err)
		return nil
	}

	return &Controller{
		cfg:      cfg,
		postConn: postConn,
		postCli:  postpb.NewPostServiceClient(postConn),
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
	loge.Errorf(ctx, "unknown protocol %v", protocol)
	return fmt.Errorf("未知的协议: %v", protocol)
}

func (c *Controller) post(ctx context.Context, req *postpb.SendTemplateRequest) error {
	var resp *postpb.SendTemplateResponse
	var err error
	utils.TimeoutOp(ctx, 10*time.Second, func(ctx context.Context) {
		resp, err = c.postCli.SendTemplate(ctx, req)
	})
	if err != nil {
		loge.Errorf(ctx, "post send template failed: %v", err)
		return err
	}
	if resp == nil || resp.Status == nil {
		err = errors.New("post send template return none")
		loge.Error(ctx, err)
		return err
	}
	if resp.Status.Status != postpb.PostStatus_PS_SUCCESS {
		err = fmt.Errorf("post send template failed: %v", resp.Status.Msg)
		loge.Error(ctx, err)
		return err
	}
	return nil
}

func (c *Controller) postSMS(ctx context.Context, purpose postsbspb.PostPurposeType,
	to, code string, expiredTimestamp int64) error {
	var templateId string
	switch purpose {
	case postsbspb.PostPurposeType_PostPurposeRegister:
		templateId = "389596"
	case postsbspb.PostPurposeType_PostPurposeLogin:
		templateId = "860253"
	case postsbspb.PostPurposeType_PostPurposeResetPassword:
		templateId = "557914"
	default:
		loge.Errorf(ctx, "unknown purpose %v", purpose)
		return fmt.Errorf("未知的: %v", purpose)
	}
	req := &postpb.SendTemplateRequest{
		ProtocolType: post.ProtocolTypeSMS,
		To:           []string{to},
		TemplateId:   templateId,
		Vars: []string{
			code,
			fmt.Sprintf("%v", int(time.Until(time.Unix(expiredTimestamp, 0)).Minutes())),
		},
	}
	return c.post(ctx, req)
}

func (c *Controller) postEmail(ctx context.Context, purpose postsbspb.PostPurposeType,
	to, code string, expiredTimestamp int64) error {
	subject := ""
	switch purpose {
	case postsbspb.PostPurposeType_PostPurposeRegister:
		subject = "羊米注册验证码"
	case postsbspb.PostPurposeType_PostPurposeLogin:
		subject = "羊米登录验证码"
	case postsbspb.PostPurposeType_PostPurposeResetPassword:
		subject = "羊米重置密码验证码"
	default:
		loge.Errorf(ctx, "unknown purpose %v", purpose)
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
