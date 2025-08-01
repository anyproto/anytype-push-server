package push

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/metric"
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
	"github.com/anyproto/anytype-push-server/repo/spacerepo"
	"github.com/anyproto/anytype-push-server/repo/spacerepo/mock_spacerepo"
	"github.com/anyproto/anytype-push-server/repo/tokenrepo"
	"github.com/anyproto/anytype-push-server/repo/tokenrepo/mock_tokenrepo"
)

var ctx = context.Background()

func TestHandler_SetToken(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		acc := newAccount()
		token := "token"
		pCtx := peer.CtxWithPeerId(ctx, "p1")
		accKey, _ := acc.GetPublic().Marshall()
		pCtx = peer.CtxWithIdentity(pCtx, accKey)

		fx.tokenRepo.EXPECT().AddToken(pCtx, domain.Token{
			Id:        token,
			AccountId: acc.GetPublic().Account(),
			PeerId:    "p1",
			Platform:  domain.PlatformAndroid,
			Status:    domain.TokenStatusValid,
		}).Return(nil)

		resp, err := fx.handler.SetToken(pCtx, &pushapi.SetTokenRequest{
			Platform: pushapi.Platform_Android,
			Token:    "token",
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
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

		req := &pushapi.SubscribeAllRequest{
			Topics: rawTopics,
		}

		ak, _ := acc.GetPublic().Marshall()
		pCtx := peer.CtxWithIdentity(ctx, ak)

		fx.accountRepo.EXPECT().SetAccountTopics(pCtx, acc.GetPublic().Account(), topics).Return(nil)

		resp, err := fx.handler.SubscribeAll(pCtx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func TestHandler_Unsubscribe(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		acc := newAccount()

		var rawTopics = &pushapi.Topics{}
		var topicsToUnsubscribe []domain.Topic

		for i := range 2 {
			rawTopic := newTopic(fmt.Sprintf("topic%d", i))
			rawTopics.Topics = append(rawTopics.Topics, rawTopic)

			topic := domain.NewTopic(rawTopic.SpaceKey, rawTopic.Topic)
			topicsToUnsubscribe = append(topicsToUnsubscribe, topic)
		}

		currentTopics := []domain.Topic{
			topicsToUnsubscribe[0],
			"otherSpaceKey/otherTopic1",
			topicsToUnsubscribe[1],
			"otherSpaceKey/otherTopic2",
		}

		expectedRemainingTopics := []domain.Topic{
			"otherSpaceKey/otherTopic1",
			"otherSpaceKey/otherTopic2",
		}

		req := &pushapi.UnsubscribeRequest{
			Topics: rawTopics,
		}

		ak, _ := acc.GetPublic().Marshall()
		pCtx := peer.CtxWithIdentity(ctx, ak)

		fx.accountRepo.EXPECT().GetTopicsByAccountId(pCtx, acc.GetPublic().Account()).Return(currentTopics, nil)
		fx.accountRepo.EXPECT().SetAccountTopics(pCtx, acc.GetPublic().Account(), expectedRemainingTopics).Return(nil)

		resp, err := fx.handler.Unsubscribe(pCtx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("unsubscribe-nonexistent-topic", func(t *testing.T) {
		fx := newFixture(t)
		acc := newAccount()

		rawTopic := newTopic("nonexistentTopic")

		req := &pushapi.UnsubscribeRequest{
			Topics: &pushapi.Topics{Topics: []*pushapi.Topic{rawTopic}},
		}

		currentTopics := []domain.Topic{
			"otherSpaceKey/otherTopic1",
			"otherSpaceKey/otherTopic2",
		}

		expectedRemainingTopics := currentTopics

		ak, _ := acc.GetPublic().Marshall()
		pCtx := peer.CtxWithIdentity(ctx, ak)

		fx.accountRepo.EXPECT().GetTopicsByAccountId(pCtx, acc.GetPublic().Account()).Return(currentTopics, nil)
		fx.accountRepo.EXPECT().SetAccountTopics(pCtx, acc.GetPublic().Account(), expectedRemainingTopics).Return(nil)

		resp, err := fx.handler.Unsubscribe(pCtx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("get-topics-error", func(t *testing.T) {
		fx := newFixture(t)
		acc := newAccount()

		rawTopic := newTopic("topicX")

		req := &pushapi.UnsubscribeRequest{
			Topics: &pushapi.Topics{Topics: []*pushapi.Topic{rawTopic}},
		}

		ak, _ := acc.GetPublic().Marshall()
		pCtx := peer.CtxWithIdentity(ctx, ak)

		fx.accountRepo.EXPECT().GetTopicsByAccountId(pCtx, acc.GetPublic().Account()).Return(nil, errors.New("db error"))

		resp, err := fx.handler.Unsubscribe(pCtx, req)
		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("empty-topics-unsubscribe", func(t *testing.T) {
		fx := newFixture(t)
		acc := newAccount()

		req := &pushapi.UnsubscribeRequest{
			Topics: &pushapi.Topics{},
		}

		ak, _ := acc.GetPublic().Marshall()
		pCtx := peer.CtxWithIdentity(ctx, ak)

		resp, err := fx.handler.Unsubscribe(pCtx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func TestHandler_Notify(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		fx := newFixture(t)
		acc := newAccount()
		rawTopic := newTopic("topicX")
		topic := domain.NewTopic(rawTopic.SpaceKey, rawTopic.Topic)
		req := newNotifyRequest(acc, []byte{1, 2, 3}, rawTopic)

		ak, _ := acc.GetPublic().Marshall()
		pCtx := peer.CtxWithIdentity(ctx, ak)

		fx.spaceRepo.EXPECT().ExistedSpaces(pCtx, []string{topic.SpaceKeyBase58()}).Return([]string{topic.SpaceKeyBase58()}, nil)
		fx.queue.EXPECT().Add(pCtx, gomock.Cond[queue.Message](func(x queue.Message) bool {
			exp := queue.Message{
				IgnoreAccountId: acc.GetPublic().Account(),
				KeyId:           req.Message.KeyId,
				Payload:         req.Message.Payload,
				Signature:       req.Message.Signature,
				Topics:          []domain.Topic{topic},
				GroupId:         "groupId",
				Silent:          false,
			}
			x.Created = time.Time{}
			return assert.Equal(t, exp, x)
		})).Return(nil)

		resp, err := fx.handler.Notify(pCtx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
	t.Run("success silent", func(t *testing.T) {
		fx := newFixture(t)
		acc := newAccount()
		rawTopic := newTopic(acc.GetPublic().Account())
		topic := domain.NewTopic(rawTopic.SpaceKey, rawTopic.Topic)

		invalidRawTopic := newTopic("1")
		invalidTopic := domain.NewTopic(invalidRawTopic.SpaceKey, invalidRawTopic.Topic)

		req := newNotifyRequest(acc, nil, rawTopic, invalidRawTopic)

		ak, _ := acc.GetPublic().Marshall()
		pCtx := peer.CtxWithIdentity(ctx, ak)

		fx.spaceRepo.EXPECT().
			ExistedSpaces(pCtx, []string{topic.SpaceKeyBase58(), invalidTopic.SpaceKeyBase58()}).
			Return([]string{topic.SpaceKeyBase58(), invalidTopic.SpaceKeyBase58()}, nil)
		fx.queue.EXPECT().Add(pCtx, gomock.Cond[queue.Message](func(x queue.Message) bool {
			exp := queue.Message{
				// expect only the valid topic where the topic field equals identity
				Topics:  []domain.Topic{topic},
				GroupId: "groupId",
				Silent:  true,
			}
			x.Created = time.Time{}
			return assert.Equal(t, exp, x)
		})).Return(nil)

		resp, err := fx.handler.NotifySilent(pCtx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func newNotifyRequest(accKey crypto.PrivKey, payload []byte, rawTopics ...*pushapi.Topic) *pushapi.NotifyRequest {
	var msg *pushapi.Message
	if payload != nil {
		sig, _ := accKey.Sign(payload)
		msg = &pushapi.Message{
			KeyId:     "key1",
			Payload:   payload,
			Signature: sig,
		}
	}
	return &pushapi.NotifyRequest{
		Topics: &pushapi.Topics{
			Topics: rawTopics,
		},
		Message: msg,
		GroupId: "groupId",
	}
}

type fixture struct {
	*push
	tokenRepo   *mock_tokenrepo.MockTokenRepo
	accountRepo *mock_accountrepo.MockAccountRepo
	spaceRepo   *mock_spacerepo.MockSpaceRepo
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
		spaceRepo:   mock_spacerepo.NewMockSpaceRepo(ctrl),
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
	fx.spaceRepo.EXPECT().Init(gomock.Any()).AnyTimes()
	fx.spaceRepo.EXPECT().Name().Return(spacerepo.CName).AnyTimes()
	fx.spaceRepo.EXPECT().Run(gomock.Any()).AnyTimes()
	fx.spaceRepo.EXPECT().Close(gomock.Any()).AnyTimes()
	fx.queue.EXPECT().Init(gomock.Any()).AnyTimes()
	fx.queue.EXPECT().Name().Return(queue.CName).AnyTimes()
	fx.queue.EXPECT().Run(gomock.Any()).AnyTimes()
	fx.queue.EXPECT().Close(gomock.Any()).AnyTimes()

	fx.a.Register(fx.tokenRepo).
		Register(fx.accountRepo).
		Register(fx.spaceRepo).
		Register(fx.queue).
		Register(metric.New()).
		Register(&testConfig{}).
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

type testConfig struct{}

func (t testConfig) Init(a *app.App) (err error) {
	return
}

func (t testConfig) Name() (name string) {
	return "config"
}

func (t testConfig) GetMetric() metric.Config {
	return metric.Config{}
}
