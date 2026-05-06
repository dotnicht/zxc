package generator

import (
	"context"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

type Generator interface {
	Post(ctx context.Context, profileName string) (string, error)
}

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "GENERATOR_PLUGIN",
	MagicCookieValue: "zxc-generator-v1",
}

var PluginMap = map[string]plugin.Plugin{
	"generator": &Plugin{},
}

type Plugin struct {
	Impl Generator
}

func (p *Plugin) Server(*plugin.MuxBroker) (any, error) {
	return &RPCServer{impl: p.Impl}, nil
}

func (p *Plugin) Client(_ *plugin.MuxBroker, c *rpc.Client) (any, error) {
	return &RPCClient{client: c}, nil
}

type RPCServer struct{ impl Generator }

func (s *RPCServer) Post(args string, resp *string) error {
	text, err := s.impl.Post(context.Background(), args)
	if err != nil {
		return err
	}
	*resp = text
	return nil
}

type RPCClient struct{ client *rpc.Client }

func (c *RPCClient) Post(_ context.Context, profileName string) (string, error) {
	var resp string
	if err := c.client.Call("Plugin.Post", profileName, &resp); err != nil {
		return "", err
	}
	return resp, nil
}
