package client

import (
	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/dapr/components-contrib/contenttype"
	"github.com/dapr/dapr/pkg/channel"
	dgrpc "github.com/dapr/dapr/pkg/grpc"
	invoke "github.com/dapr/dapr/pkg/messaging/v1"
	"github.com/dapr/dapr/pkg/modes"
	pb "github.com/dapr/dapr/pkg/proto/runtime/v1"
	"github.com/dapr/dapr/pkg/runtime"
	"github.com/dapr/go-sdk/service/common"
	"github.com/pkg/errors"
	nethttp "net/http"
)

type Config struct {
	Protocol   runtime.Protocol
	Host       string
	Port       int
	Mode       modes.DaprMode
	sslEnabled bool
}

type FuncClient struct {
	config      *Config
	ctx         *ofctx.FunctionContext
	grpc        *dgrpc.Manager
	funcChannel channel.AppChannel
}

func NewFuncClient(config *Config, ctx *ofctx.FunctionContext) *FuncClient {
	return &FuncClient{
		config: config,
		ctx:    ctx,
		grpc:   dgrpc.NewGRPCManager(modes.KubernetesMode),
	}
}

func (f *FuncClient) CreateFuncChannel() error {
	switch f.config.Protocol {
	case runtime.HTTPProtocol:
		funcChannel, err := CreateHTTPChannel(f.config.Host, f.config.Port, f.config.sslEnabled)
		if err != nil {
			return err
		}
		f.funcChannel = funcChannel
	case runtime.GRPCProtocol:
		funcChannel, err := CreateGRPCChannel(f.grpc, f.config.Host, f.config.Port, f.config.sslEnabled)
		if err != nil {
			return err
		}
		f.funcChannel = funcChannel
	default:
		return errors.Errorf("cannot create app channel for protocol %s", string(f.config.Protocol))
	}
	return nil
}

func (f *FuncClient) OnBindingEvent(ctx ofctx.Context, event *common.BindingEvent) ([]byte, error) {
	var function func(ctx ofctx.Context, event *common.BindingEvent) ([]byte, error)
	switch f.config.Protocol {
	case runtime.HTTPProtocol:
		function = f.onBindingEventHTTP
	case runtime.GRPCProtocol:
		function = f.onBindingEventGRPC
	}
	return function(ctx, event)
}

func (f *FuncClient) OnTopicEvent(ctx ofctx.Context, event *common.TopicEvent) error {
	var function func(ctx ofctx.Context, event *common.TopicEvent) error
	switch f.config.Protocol {
	case runtime.HTTPProtocol:
		function = f.onTopicEventHTTP
	case runtime.GRPCProtocol:
		function = f.onTopicEventGRPC
	}
	return function(ctx, event)
}

func (f *FuncClient) onBindingEventHTTP(ctx ofctx.Context, event *common.BindingEvent) ([]byte, error) {
	path, _ := GetComponentName(f.ctx)
	req := invoke.NewInvokeMethodRequest(path)
	req.WithHTTPExtension(nethttp.MethodPost, "")
	req.WithRawData(event.Data, invoke.JSONContentType)

	reqMetadata := map[string][]string{}
	for k, v := range event.Metadata {
		reqMetadata[k] = []string{v}
	}
	req.WithMetadata(reqMetadata)

	resp, err := f.funcChannel.InvokeMethod(ctx.GetNativeContext(), req)
	if err != nil {
		return nil, err
	}

	if resp != nil && resp.Status().Code != nethttp.StatusOK {
		return nil, errors.Errorf("Error sending binding event to function, status %d", resp.Status().Code)
	}
	_, data := resp.RawData()
	return data, nil
}

func (f *FuncClient) onBindingEventGRPC(ctx ofctx.Context, bindingEvent *common.BindingEvent) ([]byte, error) {
	client := pb.NewAppCallbackClient(f.grpc.AppClient)
	bindingName, _ := GetComponentName(f.ctx)
	req := &pb.BindingEventRequest{
		Name:     bindingName,
		Data:     bindingEvent.Data,
		Metadata: bindingEvent.Metadata,
	}
	resp, err := client.OnBindingEvent(ctx.GetNativeContext(), req)
	if err != nil {
		return resp.Data, errors.Errorf("Error sending binding event to function: %s", err)
	}
	return resp.Data, nil
}

func (f *FuncClient) onTopicEventHTTP(ctx ofctx.Context, event *common.TopicEvent) error {
	pubsubName, _ := GetComponentName(f.ctx)
	path, _ := GetTopicEventPath(f.ctx)
	req := invoke.NewInvokeMethodRequest(path)
	req.WithHTTPExtension(nethttp.MethodPost, "")
	req.WithRawData(event.RawData, contenttype.CloudEventContentType)

	metadata := make(map[string]string, 1)
	metadata["pubsubName"] = pubsubName
	req.WithCustomHTTPMetadata(metadata)

	resp, err := f.funcChannel.InvokeMethod(ctx.GetNativeContext(), req)
	if err != nil {
		return err
	}

	if resp != nil && resp.Status().Code != nethttp.StatusOK {
		return errors.Errorf("Error sending topic event to function, status %d", resp.Status().Code)
	}
	return nil
}

func (f *FuncClient) onTopicEventGRPC(ctx ofctx.Context, event *common.TopicEvent) error {
	client := pb.NewAppCallbackClient(f.grpc.AppClient)
	path, _ := GetTopicEventPath(f.ctx)
	req := &pb.TopicEventRequest{
		Id:              event.ID,
		Source:          event.Source,
		Type:            event.Type,
		SpecVersion:     event.SpecVersion,
		DataContentType: event.DataContentType,
		Data:            event.RawData,
		Topic:           event.Topic,
		PubsubName:      event.PubsubName,
		Path:            path,
	}
	_, err := client.OnTopicEvent(ctx.GetNativeContext(), req)
	if err != nil {
		return errors.Errorf("Error sending topic event to function: %s", err)
	}
	return nil
}
