package tokenrepo

import (
	"context"
	"testing"

	"github.com/anyproto/any-sync/app"
	"github.com/stretchr/testify/require"

	"github.com/anyproto/anytype-push-server/db"
	"github.com/anyproto/anytype-push-server/domain"
)

var ctx = context.Background()

func TestTokenRepo_AddToken(t *testing.T) {
	fx := newFixture(t)
	require.NoError(t, fx.AddToken(ctx, domain.Token{
		Id:        "1",
		AccountId: "a",
		PeerId:    "122",
		Platform:  domain.PlatformAndroid,
	}))
	require.NoError(t, fx.AddToken(ctx, domain.Token{
		Id:        "1",
		AccountId: "b",
		PeerId:    "122",
		Platform:  domain.PlatformAndroid,
	}))
	tokens, err := fx.GetActiveTokensByAccountIds(ctx, []string{"b"})
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	require.Equal(t, "1", tokens[0].Id)
}

func TestTokenRepo_UpdateTokenStatus(t *testing.T) {
	fx := newFixture(t)
	require.NoError(t, fx.AddToken(ctx, domain.Token{
		Id:        "1",
		AccountId: "a",
		PeerId:    "122",
		Platform:  domain.PlatformAndroid,
	}))
	require.NoError(t, fx.UpdateTokenStatus(ctx, "1", domain.TokenStatusInvalid))
	tokens, err := fx.GetActiveTokensByAccountIds(ctx, []string{"a"})
	require.NoError(t, err)
	require.Len(t, tokens, 0)
}

func newFixture(t testing.TB) *fixture {
	fx := &fixture{
		TokenRepo: New(),
		a:         new(app.App),
	}
	fx.a.Register(&testConfig{
		Mongo: db.Mongo{
			Connect:  "mongodb://localhost:27017",
			Database: "publish_unittest",
		},
	}).
		Register(db.New()).
		Register(fx.TokenRepo)
	require.NoError(t, fx.a.Start(ctx))
	t.Cleanup(func() {
		fx.finish(t)
	})
	return fx
}

type fixture struct {
	TokenRepo
	a *app.App
}

func (fx *fixture) finish(t testing.TB) {
	_ = fx.TokenRepo.(*tokenRepo).coll.Drop(ctx)
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
