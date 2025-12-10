package spacerepo

import (
	"context"
	"testing"

	"github.com/anyproto/any-sync/app"
	"github.com/stretchr/testify/require"

	"github.com/anyproto/anytype-push-server/db"
	"github.com/anyproto/anytype-push-server/domain"
)

var ctx = context.Background()

func TestSpaceRepo_Create(t *testing.T) {
	fx := newFixture(t)
	require.NoError(t, fx.Create(ctx, domain.Space{
		Id:     "1",
		Author: "a",
	}))
	require.ErrorIs(t, fx.Create(ctx, domain.Space{
		Id:     "1",
		Author: "b",
	}), ErrSpaceExists)
}

func TestSpaceRepo_ExistedSpaces(t *testing.T) {
	t.Skip()
	fx := newFixture(t)
	require.NoError(t, fx.Create(ctx, domain.Space{
		Id:     "1",
		Author: "a",
	}))
	require.NoError(t, fx.Create(ctx, domain.Space{
		Id:     "2",
		Author: "a",
	}))
	result, err := fx.ExistedSpaces(ctx, []string{"1", "2", "3"})
	require.NoError(t, err)
	require.Equal(t, []string{"1", "2"}, result)
}

func TestSpaceRepo_Remove(t *testing.T) {
	fx := newFixture(t)
	require.NoError(t, fx.Create(ctx, domain.Space{
		Id:     "1",
		Author: "a",
	}))
	require.NoError(t, fx.Remove(ctx, domain.Space{
		Id:     "1",
		Author: "a",
	}))
	require.ErrorIs(t, fx.Remove(ctx, domain.Space{
		Id:     "1",
		Author: "a",
	}), ErrSpaceNotFound)
}

func newFixture(t testing.TB) *fixture {
	fx := &fixture{
		SpaceRepo: New(),
		a:         new(app.App),
	}
	fx.a.Register(&testConfig{
		Mongo: db.Mongo{
			Connect:  "mongodb://localhost:27017",
			Database: "publish_unittest",
		},
	}).
		Register(db.New()).
		Register(fx.SpaceRepo)
	require.NoError(t, fx.a.Start(ctx))
	t.Cleanup(func() {
		fx.finish(t)
	})
	return fx
}

type fixture struct {
	SpaceRepo
	a *app.App
}

func (fx *fixture) finish(t testing.TB) {
	_ = fx.SpaceRepo.(*spaceRepo).coll.Drop(ctx)
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
