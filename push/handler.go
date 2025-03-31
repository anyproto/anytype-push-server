package push

import (
	"context"
	"time"

	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/peer"
	"go.uber.org/zap"

	"github.com/anyproto/anytype-push-server/pushclient/pushapi"
)

var _ pushapi.DRPCPushServer = (*handler)(nil)

type handler struct {
	p *push
}

func (h *handler) CreateSpace(ctx context.Context, req *pushapi.CreateSpaceRequest) (resp *pushapi.Ok, err error) {
	st := time.Now()
	defer func() {
		h.p.metric.RequestLog(ctx, "push.createSpace",
			metric.TotalDur(time.Since(st)),
			zap.String("addr", peer.CtxPeerAddr(ctx)),
			zap.Error(err),
		)
	}()
	if err = h.p.CreateSpace(ctx, req.SpaceKey, req.AccountSignature); err != nil {
		return
	}
	return &pushapi.Ok{}, nil
}

func (h *handler) RemoveSpace(ctx context.Context, req *pushapi.RemoveSpaceRequest) (resp *pushapi.Ok, err error) {
	st := time.Now()
	defer func() {
		h.p.metric.RequestLog(ctx, "push.removeSpace",
			metric.TotalDur(time.Since(st)),
			zap.String("addr", peer.CtxPeerAddr(ctx)),
			zap.Error(err),
		)
	}()
	if err = h.p.RemoveSpace(ctx, req.SpaceKey, req.AccountSignature); err != nil {
		return
	}
	return &pushapi.Ok{}, nil
}

func (h *handler) Subscriptions(ctx context.Context, req *pushapi.SubscriptionsRequest) (resp *pushapi.SubscriptionsResponse, err error) {
	st := time.Now()
	defer func() {
		h.p.metric.RequestLog(ctx, "push.subscriptions",
			metric.TotalDur(time.Since(st)),
			zap.String("addr", peer.CtxPeerAddr(ctx)),
			zap.Error(err),
		)
	}()
	topics, err := h.p.Subscriptions(ctx)
	if err != nil {
		return
	}
	return &pushapi.SubscriptionsResponse{
		Topics: topics,
	}, nil
}

func (h *handler) Subscribe(ctx context.Context, req *pushapi.SubscribeRequest) (resp *pushapi.Ok, err error) {
	st := time.Now()
	defer func() {
		h.p.metric.RequestLog(ctx, "push.subscribe",
			metric.TotalDur(time.Since(st)),
			zap.String("addr", peer.CtxPeerAddr(ctx)),
			zap.Error(err),
		)
	}()
	if err = h.p.Subscribe(ctx, req.Topics); err != nil {
		return
	}
	return &pushapi.Ok{}, nil
}

func (h *handler) Unsubscribe(ctx context.Context, req *pushapi.UnsubscribeRequest) (resp *pushapi.Ok, err error) {
	st := time.Now()
	defer func() {
		h.p.metric.RequestLog(ctx, "push.unsubscribe",
			metric.TotalDur(time.Since(st)),
			zap.String("addr", peer.CtxPeerAddr(ctx)),
			zap.Error(err),
		)
	}()
	if err = h.p.Unsubscribe(ctx, req.Topics); err != nil {
		return
	}
	return &pushapi.Ok{}, nil
}

func (h *handler) SetToken(ctx context.Context, req *pushapi.SetTokenRequest) (resp *pushapi.Ok, err error) {
	st := time.Now()
	defer func() {
		h.p.metric.RequestLog(ctx, "push.setToken",
			metric.TotalDur(time.Since(st)),
			zap.String("addr", peer.CtxPeerAddr(ctx)),
			zap.Error(err),
		)
	}()
	if err = h.p.AddToken(ctx, req); err != nil {
		return
	}
	return &pushapi.Ok{}, nil
}

func (h *handler) SubscribeAll(ctx context.Context, req *pushapi.SubscribeAllRequest) (resp *pushapi.Ok, err error) {
	st := time.Now()
	defer func() {
		h.p.metric.RequestLog(ctx, "push.subscribeAll",
			metric.TotalDur(time.Since(st)),
			zap.String("addr", peer.CtxPeerAddr(ctx)),
			zap.Error(err),
		)
	}()
	if err = h.p.SubscribeAll(ctx, req); err != nil {
		return
	}
	return &pushapi.Ok{}, nil
}

func (h *handler) Notify(ctx context.Context, req *pushapi.NotifyRequest) (resp *pushapi.Ok, err error) {
	st := time.Now()
	defer func() {
		h.p.metric.RequestLog(ctx, "push.notify",
			metric.TotalDur(time.Since(st)),
			zap.String("addr", peer.CtxPeerAddr(ctx)),
			zap.Error(err),
		)
	}()
	if err = h.p.Notify(ctx, req); err != nil {
		return
	}
	return &pushapi.Ok{}, nil
}
