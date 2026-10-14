package sender

import (
	"context"
	"sync"
	"testing"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/anyproto/anytype-push-server/domain"
	"github.com/anyproto/anytype-push-server/queue"
	"github.com/anyproto/anytype-push-server/queue/mock_queue"
	"github.com/anyproto/anytype-push-server/repo/accountrepo"
	"github.com/anyproto/anytype-push-server/repo/accountrepo/mock_accountrepo"
	"github.com/anyproto/anytype-push-server/repo/tokenrepo"
	"github.com/anyproto/anytype-push-server/repo/tokenrepo/mock_tokenrepo"
)

var ctx = context.Background()

func TestSender_SendMessage(t *testing.T) {
	topics := []domain.Topic{"space/topic"}
	// fresh slice per call: the sender appends to the slice it gets
	tokens := func() []domain.Token {
		return []domain.Token{
			{Id: "a1", AccountId: "A", PeerId: "pA1", Platform: domain.PlatformAndroid},
			{Id: "a2", AccountId: "A", PeerId: "pA2", Platform: domain.PlatformIOS},
			{Id: "b1", AccountId: "B", PeerId: "pB1", Platform: domain.PlatformAndroid},
		}
	}

	t.Run("ignore account", func(t *testing.T) {
		fx := newFixture(t)
		fx.accountRepo.EXPECT().GetAccountIdsByTopics(gomock.Any(), topics).Return([]string{"A", "B"}, nil)
		fx.tokenRepo.EXPECT().GetActiveTokensByAccountIds(gomock.Any(), []string{"B"}).Return(tokens()[2:], nil)

		require.NoError(t, fx.SendMessage(queue.Message{
			IgnoreAccountId: "A",
			Topics:          topics,
			GroupId:         "g",
		}))
		assert.Equal(t, []string{"b1"}, fx.provider.tokens(domain.PlatformAndroid))
		assert.Nil(t, fx.provider.tokens(domain.PlatformIOS))
	})

	t.Run("no exclusion", func(t *testing.T) {
		fx := newFixture(t)
		fx.accountRepo.EXPECT().GetAccountIdsByTopics(gomock.Any(), topics).Return([]string{"A", "B"}, nil)
		fx.tokenRepo.EXPECT().GetActiveTokensByAccountIds(gomock.Any(), []string{"A", "B"}).Return(tokens(), nil)

		// every token of every matched account, the publisher's included
		require.NoError(t, fx.SendMessage(queue.Message{
			Topics:    topics,
			GroupId:   "g",
			Payload:   []byte{1},
			Signature: []byte{2},
			KeyId:     "k",
		}))
		assert.Equal(t, []string{"a1", "b1"}, fx.provider.tokens(domain.PlatformAndroid))
		assert.Equal(t, []string{"a2"}, fx.provider.tokens(domain.PlatformIOS))
		msg := fx.provider.messages[domain.PlatformIOS][0]
		assert.False(t, msg.Silent)
		assert.Equal(t, "normal", msg.Data["x-any-type"])
		assert.Equal(t, "g", msg.Data["x-any-group-id"])
		assert.Equal(t, "k", msg.Data["x-any-key-id"])
	})

	t.Run("silent", func(t *testing.T) {
		fx := newFixture(t)
		fx.accountRepo.EXPECT().GetAccountIdsByTopics(gomock.Any(), topics).Return([]string{"A"}, nil)
		fx.tokenRepo.EXPECT().GetActiveTokensByAccountIds(gomock.Any(), []string{"A"}).Return(tokens()[:2], nil)

		// the silent path excludes nothing: every own device wakes
		require.NoError(t, fx.SendMessage(queue.Message{
			Topics: topics,
			Silent: true,
		}))
		assert.Equal(t, []string{"a1"}, fx.provider.tokens(domain.PlatformAndroid))
		assert.Equal(t, []string{"a2"}, fx.provider.tokens(domain.PlatformIOS))
		msg := fx.provider.messages[domain.PlatformIOS][0]
		assert.True(t, msg.Silent)
		assert.Equal(t, "silent", msg.Data["x-any-type"])
	})
}

type fakeProvider struct {
	mu       sync.Mutex
	messages map[domain.Platform][]domain.Message
}

func (f *fakeProvider) SendMessage(_ context.Context, message domain.Message, _ func(token string)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.messages[message.Platform] = append(f.messages[message.Platform], message)
	return nil
}

func (f *fakeProvider) tokens(p domain.Platform) (res []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range f.messages[p] {
		res = append(res, m.Tokens...)
	}
	return
}

type fixture struct {
	*sender
	tokenRepo   *mock_tokenrepo.MockTokenRepo
	accountRepo *mock_accountrepo.MockAccountRepo
	queue       *mock_queue.MockQueue
	provider    *fakeProvider
	a           *app.App
}

func newFixture(t *testing.T) *fixture {
	ctrl := gomock.NewController(t)
	fx := &fixture{
		sender:      New().(*sender),
		a:           new(app.App),
		tokenRepo:   mock_tokenrepo.NewMockTokenRepo(ctrl),
		accountRepo: mock_accountrepo.NewMockAccountRepo(ctrl),
		queue:       mock_queue.NewMockQueue(ctrl),
		provider:    &fakeProvider{messages: map[domain.Platform][]domain.Message{}},
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
	fx.queue.EXPECT().Consume(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	fx.queue.EXPECT().Close(gomock.Any()).AnyTimes()

	fx.a.Register(fx.tokenRepo).
		Register(fx.accountRepo).
		Register(fx.queue).
		Register(metric.New()).
		Register(&testConfig{}).
		Register(fx.sender)
	require.NoError(t, fx.a.Start(ctx))
	fx.RegisterProvider(domain.PlatformAndroid, fx.provider)
	fx.RegisterProvider(domain.PlatformIOS, fx.provider)
	t.Cleanup(func() {
		require.NoError(t, fx.a.Close(ctx))
		ctrl.Finish()
	})
	return fx
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
