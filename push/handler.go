package push

import (
	"context"

	"github.com/anyproto/anytype-push-server/pushclient/pushapi"
)

type handler struct {
	p *push
}

func (h *handler) SetToken(ctx context.Context, req *pushapi.SetTokenRequest) (resp *pushapi.Ok, err error) {
	if err = h.p.AddToken(ctx, req); err != nil {
		return
	}
	return &pushapi.Ok{}, nil
}

func (h *handler) SubscribeAll(ctx context.Context, req *pushapi.SubscribeAllRequest) (resp *pushapi.Ok, err error) {
	if err = h.p.SubscribeAll(ctx, req); err != nil {
		return
	}
	return &pushapi.Ok{}, nil
}

func (h *handler) Notify(ctx context.Context, req *pushapi.NotifyRequest) (resp *pushapi.Ok, err error) {
	if err = h.p.Notify(ctx, req); err != nil {
		return
	}
	return &pushapi.Ok{}, nil
}
