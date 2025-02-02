package push

import (
	"context"
	"fmt"
	"testing"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/rpc/rpctest"
	"github.com/anyproto/any-sync/util/crypto"
	"github.com/mr-tron/base58"

	"github.com/anyproto/any-sync/testutil/accounttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/anyproto/anytype-push-server/domain"
	"github.com/anyproto/anytype-push-server/pushclient/pushapi"
	"github.com/anyproto/anytype-push-server/queue"
	"github.com/anyproto/anytype-push-server/queue/mock_queue"
	"github.com/anyproto/anytype-push-server/repo/accountrepo"
	"github.com/anyproto/anytype-push-server/repo/accountrepo/mock_accountrepo"
	"github.com/anyproto/anytype-push-server/repo/tokenrepo"
	"github.com/anyproto/anytype-push-server/repo/tokenrepo/mock_tokenrepo"
)

var ctx = context.Background()

func TestHandler_SetToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		acc := newAccount()
		token := "token"
		sign, err := acc.Sign([]byte(token))
		require.NoError(t, err)
		pCtx := peer.CtxWithPeerId(ctx, "p1")

		fx.tokenRepo.EXPECT().AddToken(pCtx, domain.Token{
			Id:        token,
			AccountId: acc.GetPublic().Account(),
			PeerId:    "p1",
			Platform:  domain.PlatformAndroid,
			Status:    domain.TokenStatusValid,
		}).Return(nil)

		resp, err := fx.handler.SetToken(pCtx, &pushapi.SetTokenRequest{
			AccountId: acc.GetPublic().Account(),
			Platform:  pushapi.Platform_Android,
			Token:     "token",
			Signature: sign,
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
	t.Run("invalid signature", func(t *testing.T) {
		fx := newFixture(t)
		acc := newAccount()
		token := "token"
		sign, err := acc.Sign([]byte(token + "invalid"))
		require.NoError(t, err)
		pCtx := peer.CtxWithPeerId(ctx, "p1")

		resp, err := fx.handler.SetToken(pCtx, &pushapi.SetTokenRequest{
			AccountId: acc.GetPublic().Account(),
			Platform:  pushapi.Platform_Android,
			Token:     "token",
			Signature: sign,
		})
		require.ErrorIs(t, err, pushapi.ErrInvalidSignature)
		assert.Nil(t, resp)
	})
}

func TestHandler_SubscribeAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		acc := newAccount()
		var rawTopics = &pushapi.Topics{}
		var topics []domain.Topic
		for i := range 2 {
			rawTopic := newTopic(fmt.Sprintf("topic%d", i))
			rawTopics.Topics = append(rawTopics.Topics, rawTopic)
			topics = append(topics, domain.Topic(base58.Encode(rawTopic.SpaceKey)+"/"+rawTopic.Topic))
		}
		rawTopicsData, err := rawTopics.Marshal()
		require.NoError(t, err)

		signature, err := acc.Sign(rawTopicsData)
		require.NoError(t, err)

		req := &pushapi.SubscribeAllRequest{
			AccountId: acc.GetPublic().Account(),
			Payload:   rawTopicsData,
			Signature: signature,
		}

		fx.accountRepo.EXPECT().SetAccountTopics(ctx, req.AccountId, topics).Return(nil)

		resp, err := fx.handler.SubscribeAll(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		t.Log(topics)
	})
}

type fixture struct {
	*push
	tokenRepo   *mock_tokenrepo.MockTokenRepo
	accountRepo *mock_accountrepo.MockAccountRepo
	queue       *mock_queue.MockQueue
	a           *app.App
}

func newFixture(t *testing.T) *fixture {
	ctrl := gomock.NewController(t)
	fx := &fixture{
		push:        New().(*push),
		a:           new(app.App),
		tokenRepo:   mock_tokenrepo.NewMockTokenRepo(ctrl),
		accountRepo: mock_accountrepo.NewMockAccountRepo(ctrl),
		queue:       mock_queue.NewMockQueue(ctrl),
	}
	fx.tokenRepo.EXPECT().Name().Return(tokenrepo.CName).AnyTimes()
	fx.tokenRepo.EXPECT().Init(gomock.Any()).AnyTimes()
	fx.tokenRepo.EXPECT().Run(gomock.Any()).AnyTimes()
	fx.tokenRepo.EXPECT().Close(gomock.Any()).AnyTimes()
	fx.accountRepo.EXPECT().Init(gomock.Any()).AnyTimes()
	fx.accountRepo.EXPECT().Name().Return(accountrepo.CName).AnyTimes()
	fx.accountRepo.EXPECT().Run(gomock.Any()).AnyTimes()
	fx.accountRepo.EXPECT().Close(gomock.Any()).AnyTimes()
	fx.queue.EXPECT().Init(gomock.Any()).AnyTimes()
	fx.queue.EXPECT().Name().Return(queue.CName).AnyTimes()
	fx.queue.EXPECT().Run(gomock.Any()).AnyTimes()
	fx.queue.EXPECT().Close(gomock.Any()).AnyTimes()

	fx.a.Register(fx.tokenRepo).
		Register(fx.accountRepo).
		Register(fx.queue).
		Register(fx.push).
		Register(rpctest.NewTestServer())
	require.NoError(t, fx.a.Start(ctx))
	t.Cleanup(func() {
		require.NoError(t, fx.a.Close(ctx))
		ctrl.Finish()
	})
	return fx
}

func newAccount() crypto.PrivKey {
	as := accounttest.AccountTestService{}
	_ = as.Init(nil)
	return as.Account().SignKey
}

func newTopic(topic string) *pushapi.Topic {
	privKey, pubKey, _ := crypto.GenerateRandomEd25519KeyPair()
	signature, _ := privKey.Sign([]byte(topic))
	rawPubKey, _ := pubKey.Raw()
	return &pushapi.Topic{
		SpaceKey:  rawPubKey,
		Topic:     topic,
		Signature: signature,
	}
}
