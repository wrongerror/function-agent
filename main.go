package main

import (
	"context"
	"github.com/dapr/dapr/pkg/modes"
	"github.com/dapr/dapr/pkg/runtime"
	"k8s.io/klog/v2"
	"os"
	"strconv"

	"github.com/OpenFunction/function-agent/client"

	ofctx "github.com/OpenFunction/functions-framework-go/context"
	"github.com/OpenFunction/functions-framework-go/framework"
)

const (
	defaultAppProtocol = "grpc"
	protocolEnvVar     = "App_Protocol"
)

var (
	funcClient *client.FuncClient
)

func main() {
	ctx := context.Background()
	fwk, err := framework.NewFramework()
	if err != nil {
		klog.Exit(err)
	}

	funcContext := client.GetFuncContext(fwk)

	host := client.GetFuncHost(funcContext)
	port, _ := strconv.Atoi(funcContext.GetPort())
	protocol := os.Getenv(protocolEnvVar)
	if protocol == "" {
		protocol = defaultAppProtocol
	}
	config := &client.Config{
		Protocol: runtime.Protocol(protocol),
		Host:     host,
		Port:     port,
		Mode:     modes.KubernetesMode,
	}

	funcClient = client.NewFuncClient(config, funcContext)
	if err := funcClient.CreateFuncChannel(); err != nil {
		klog.Exit(err)
	}

	if err := fwk.Register(ctx, EventHandler); err != nil {
		klog.Exit(err)
	}

	if err := fwk.Start(ctx); err != nil {
		klog.Exit(err)
	}
}

func EventHandler(ctx ofctx.Context, in []byte) (ofctx.Out, error) {
	// Forwarding BindingEvent
	bindingEvent := ctx.GetBindingEvent()
	if bindingEvent != nil {
		if data, err := funcClient.OnBindingEvent(ctx, bindingEvent); err != nil {
			klog.Error(err)
			out := new(ofctx.FunctionOut)
			out.WithData(data)
			out.WithCode(ofctx.Success)
			return out, nil
		} else {
			return ctx.ReturnOnSuccess(), err
		}
	}

	// Forwarding TopicEvent
	topicEvent := ctx.GetTopicEvent()
	if topicEvent != nil {
		if err := funcClient.OnTopicEvent(ctx, topicEvent); err != nil {
			klog.Error(err)
			out := new(ofctx.FunctionOut)
			out.WithCode(ofctx.Success)
			return out, nil
		} else {
			return ctx.ReturnOnSuccess(), err
		}
	}

	return ctx.ReturnOnInternalError(), nil
}
