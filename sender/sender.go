package sender

import (
	"context"
	"encoding/base64"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/metric"
	"github.com/cheggaaa/mb/v3"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/anyproto/anytype-push-server/domain"
	"github.com/anyproto/anytype-push-server/queue"
	"github.com/anyproto/anytype-push-server/repo/accountrepo"
	"github.com/anyproto/anytype-push-server/repo/tokenrepo"
)

const CName = "push.sender"

var log = logger.NewNamed(CName)

func New() Sender {
	return new(sender)
}

type Sender interface {
	RegisterProvider(p domain.Platform, provider Provider)
	app.ComponentRunnable
}

type Provider interface {
	SendMessage(ctx context.Context, message domain.Message, onInvalid func(token string)) (err error)
}

type sender struct {
	accountRepo   accountrepo.AccountRepo
	tokenRepo     tokenrepo.TokenRepo
	queue         queue.Queue
	invalidTokens *mb.MB[string]
	providers     map[domain.Platform]Provider
	metrics       struct {
		sendTokens   atomic.Uint64
		errorTokens  atomic.Uint64
		sendCount    atomic.Uint64
		sendDuration *prometheus.SummaryVec
	}
}

func (s *sender) Init(a *app.App) (err error) {
	s.accountRepo = a.MustComponent(accountrepo.CName).(accountrepo.AccountRepo)
	s.tokenRepo = a.MustComponent(tokenrepo.CName).(tokenrepo.TokenRepo)
	s.queue = a.MustComponent(queue.CName).(queue.Queue)
	s.providers = make(map[domain.Platform]Provider)
	s.invalidTokens = mb.New[string](100)
	registerMetrics(a.MustComponent(metric.CName).(metric.Metric).Registry(), s)
	return
}

func (s *sender) Name() (name string) {
	return CName
}

func (s *sender) Run(ctx context.Context) (err error) {
	go s.removeTokensBatch()
	// TODO: move the num runners to the config
	for range 10 {
		if err = s.queue.Consume(ctx, s.SendMessage); err != nil {
			return
		}
	}
	return
}

func (s *sender) RegisterProvider(p domain.Platform, provider Provider) {
	s.providers[p] = provider
}

func (s *sender) SendMessage(message queue.Message) (err error) {
	ctx := context.Background()
	accountIds, err := s.accountRepo.GetAccountIdsByTopics(ctx, message.Topics)
	if err != nil {
		return
	}
	accountIds = slices.DeleteFunc(accountIds, func(s string) bool {
		return s == message.IgnoreAccountId
	})
	tokens, err := s.tokenRepo.GetActiveTokensByAccountIds(ctx, accountIds)
	if err != nil {
		return
	}
	if len(tokens) == 0 {
		return
	}

	data := make(map[string]string)

	data["x-any-key-id"] = message.KeyId
	data["x-any-payload"] = base64.StdEncoding.EncodeToString(message.Payload)
	data["x-any-signature"] = base64.StdEncoding.EncodeToString(message.Signature)

	var byProvider = make(map[domain.Platform]*domain.Message)

	for _, token := range tokens {
		msg := byProvider[token.Platform]
		if msg == nil {
			msg = &domain.Message{
				Platform: token.Platform,
				Tokens:   []string{token.Id},
				Data:     data,
			}
		} else {
			msg.Tokens = append(msg.Tokens, token.Id)
		}
		byProvider[token.Platform] = msg
	}

	for prv, msg := range byProvider {
		provider, ok := s.providers[prv]
		if !ok {
			log.Warn("unexpected provider", zap.String("provider", fmt.Sprint(prv)))
		} else {
			if err = provider.SendMessage(ctx, *msg, s.onInvalid); err != nil {
				return err
			}
			s.metrics.sendCount.Add(1)
			s.metrics.sendTokens.Add(uint64(len(msg.Tokens)))
			dur := time.Since(message.Created)
			s.metrics.sendDuration.WithLabelValues(prv.String()).Observe(dur.Seconds())
		}
	}
	return nil
}

func (s *sender) onInvalid(token string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = s.invalidTokens.Add(ctx, token)
}

func (s *sender) removeTokensBatch() {
	ctx := mb.CtxWithTimeLimit(context.Background(), time.Second)
	cond := s.invalidTokens.NewCond().WithMin(10)
	for {
		tokens, err := cond.Wait(ctx)
		if err != nil {
			return
		}
		st := time.Now()
		s.metrics.errorTokens.Add(uint64(len(tokens)))
		if err = s.tokenRepo.RemoveTokens(ctx, tokens); err != nil {
			log.Error("remove tokens error", zap.Error(err))
		} else {
			log.Info("remove tokens success", zap.Int("count", len(tokens)), zap.Duration("dur", time.Since(st)))
		}
	}
}

func (s *sender) Close(ctx context.Context) (err error) {
	return s.invalidTokens.Close()
}
