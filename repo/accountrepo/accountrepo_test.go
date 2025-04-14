package accountrepo

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/anyproto/any-sync/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anyproto/anytype-push-server/db"
	"github.com/anyproto/anytype-push-server/domain"
)

var ctx = context.Background()

func newTestTopic() domain.Topic {
	var spaceKey = make([]byte, 32)
	_, _ = rand.Read(spaceKey)
	return domain.NewTopic(spaceKey, "topic")
}

func TestAccountRepo_SetAccountTopics(t *testing.T) {
	fx := newFixture(t)
	topics := []domain.Topic{newTestTopic(), newTestTopic(), newTestTopic()}
	require.NoError(t, fx.SetAccountTopics(ctx, "a", topics))
	require.NoError(t, fx.SetAccountTopics(ctx, "a", topics[:2]))
}

func TestAccountRepo_GetAccountTopics(t *testing.T) {
	fx := newFixture(t)
	topics := []domain.Topic{newTestTopic(), newTestTopic(), newTestTopic()}
	require.NoError(t, fx.SetAccountTopics(ctx, "a", topics[:2]))
	require.NoError(t, fx.SetAccountTopics(ctx, "b", topics[1:]))

	result, err := fx.GetAccountIdsByTopics(ctx, topics)
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Contains(t, result, "a")
	assert.Contains(t, result, "b")

	result, err = fx.GetAccountIdsByTopics(ctx, topics[2:])
	require.NoError(t, err)
	assert.Equal(t, []string{"b"}, result)

	result, err = fx.GetAccountIdsByTopics(ctx, topics[:1])
	require.NoError(t, err)
	assert.Equal(t, []string{"a"}, result)
}

func TestAccountRepo_GetTopicsByAccount(t *testing.T) {
	fx := newFixture(t)
	topics := []domain.Topic{newTestTopic(), newTestTopic(), newTestTopic()}
	require.NoError(t, fx.SetAccountTopics(ctx, "a", topics[:1]))
	require.NoError(t, fx.SetAccountTopics(ctx, "b", topics[1:]))

	result, err := fx.GetTopicsByAccountId(ctx, "a")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Contains(t, result, topics[0])

	result, err = fx.GetTopicsByAccountId(ctx, "b")
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Contains(t, result, topics[1])
	assert.Contains(t, result, topics[2])

}

func newFixture(t testing.TB) *fixture {
	fx := &fixture{
		AccountRepo: New(),
		a:           new(app.App),
	}
	fx.a.Register(&testConfig{
		Mongo: db.Mongo{
			Connect:  "mongodb://localhost:27017",
			Database: "publish_unittest",
		},
	}).
		Register(db.New()).
		Register(fx.AccountRepo)
	require.NoError(t, fx.a.Start(ctx))
	t.Cleanup(func() {
		fx.finish(t)
	})
	return fx
}

type fixture struct {
	AccountRepo
	a *app.App
}

func (fx *fixture) finish(t testing.TB) {
	_ = fx.AccountRepo.(*accountRepo).coll.Drop(ctx)
	require.NoError(t, fx.a.Close(ctx))
}

type testConfig struct {
	Mongo db.Mongo
}

func (t testConfig) Init(a *app.App) (err error) {
	return
}

func (t testConfig) Name() (name string) {
	return "config"
}

func (t testConfig) GetMongo() db.Mongo {
	return t.Mongo
}
