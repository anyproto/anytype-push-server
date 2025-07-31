package queue

import (
	"context"
	"testing"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/testutil/accounttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anyproto/anytype-push-server/domain"
	"github.com/anyproto/anytype-push-server/redisprovider/testredisprovider"
)

var ctx = context.Background()

func TestQueue_Consume(t *testing.T) {
	fx := newFixture(t)
	var toSend = []Message{
		{Topics: []domain.Topic{"1"}, Created: time.Now().Round(time.Hour)},
		{Topics: []domain.Topic{"2"}, Created: time.Now().Round(time.Hour)},
	}
	require.NoError(t, fx.Add(ctx, toSend[0]))
	var msgs = make(chan Message)
	require.NoError(t, fx.Consume(ctx, func(msg Message) error {
		msgs <- msg
		return nil
	}))

	require.NoError(t, fx.Add(ctx, toSend[1]))
	var result = make([]Message, 2)
	for i := range result {
		select {
		case msg := <-msgs:
			result[i] = msg
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
	assert.Equal(t, toSend, result)
}

type fixture struct {
	Queue
	a *app.App
}

func newFixture(t *testing.T) *fixture {
	fx := &fixture{
		Queue: New(),
		a:     new(app.App),
	}
	fx.a.Register(&accounttest.AccountTestService{}).Register(testredisprovider.NewTestRedisProvider()).Register(fx.Queue)
	require.NoError(t, fx.a.Start(ctx))
	t.Cleanup(func() {
		require.NoError(t, fx.a.Close(ctx))
	})
	return fx
}
