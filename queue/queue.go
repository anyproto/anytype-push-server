//go:generate mockgen -destination mock_queue/mock_queue.go github.com/anyproto/anytype-push-server/queue Queue

package queue

import (
	"context"
	"encoding/json"
	"time"

	"github.com/adjust/rmq/v5"
	"github.com/anyproto/any-sync/accountservice"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/anyproto/anytype-push-server/domain"
	"github.com/anyproto/anytype-push-server/redisprovider"
)

const CName = "push.queue"

var log = logger.NewNamed(CName)

func New() Queue {
	return new(queue)
}

type Message struct {
	IgnoreAccountId string         `json:"ignoreAccountId"`
	Topics          []domain.Topic `json:"topics"`
}

type Queue interface {
	Add(ctx context.Context, msg Message) error
	Consume(ctx context.Context, handle func(msg Message) error) error
	app.ComponentRunnable
}

type queue struct {
	client       redis.UniversalClient
	rmqConn      rmq.Connection
	queue        rmq.Queue
	errCh        chan error
	accountId    string
	runCtx       context.Context
	runCtxCancel context.CancelFunc
}

func (q *queue) Init(a *app.App) (err error) {
	q.client = a.MustComponent(redisprovider.CName).(redisprovider.RedisProvider).Redis()
	q.accountId = a.MustComponent(accountservice.CName).(accountservice.Service).Account().SignKey.GetPublic().Account()
	q.runCtx, q.runCtxCancel = context.WithCancel(context.Background())
	return
}

func (q *queue) Name() (name string) {
	return CName
}

func (q *queue) Run(ctx context.Context) (err error) {
	q.errCh = make(chan error, 10)
	if q.rmqConn, err = rmq.OpenClusterConnection(q.accountId, q.client, q.errCh); err != nil {
		return err
	}
	go q.handleRmqErrs()
	if q.queue, err = q.rmqConn.OpenQueue("msgs"); err != nil {
		return err
	}
	return q.queue.StartConsuming(10, time.Millisecond*100)
}

func (q *queue) Add(ctx context.Context, msg Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return q.queue.Publish(string(data))
}

func (q *queue) Consume(ctx context.Context, handle func(msg Message) error) error {
	cons := func(delivery rmq.Delivery) {
		select {
		case <-q.runCtx.Done():
			_ = delivery.Reject()
		case <-ctx.Done():
			_ = delivery.Reject()
		default:
		}
		var msg Message
		if err := json.Unmarshal([]byte(delivery.Payload()), &msg); err != nil {
			_ = delivery.Reject()
			return
		}
		err := handle(msg)
		if err != nil {
			_ = delivery.Reject()
		} else {
			_ = delivery.Ack()
		}
	}
	_, err := q.queue.AddConsumerFunc(q.accountId, cons)
	return err
}

func (q *queue) handleRmqErrs() {
	for {
		select {
		case <-q.runCtx.Done():
			return
		case err := <-q.errCh:
			log.Warn("rmq error", zap.Error(err))
		}
	}
}

func (q *queue) Close(ctx context.Context) (err error) {
	if q.runCtxCancel != nil {
		q.runCtxCancel()
	}
	if q.queue != nil {
		done := q.queue.StopConsuming()
		<-done
	}
	return nil
}
