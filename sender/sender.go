package sender

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
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
	accountRepo accountrepo.AccountRepo
	tokenRepo   tokenrepo.TokenRepo
	queue       queue.Queue
	providers   map[domain.Platform]Provider
}

func (s *sender) Init(a *app.App) (err error) {
	s.accountRepo = a.MustComponent(accountrepo.CName).(accountrepo.AccountRepo)
	s.tokenRepo = a.MustComponent(tokenrepo.CName).(tokenrepo.TokenRepo)
	s.queue = a.MustComponent(queue.CName).(queue.Queue)
	s.providers = make(map[domain.Platform]Provider)
	return
}

func (s *sender) Name() (name string) {
	return CName
}

func (s *sender) Run(ctx context.Context) (err error) {
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
	topicsJSON, _ := json.Marshal(message.Topics)
	data["x-anytype-topics"] = string(topicsJSON)

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
		}
	}
	return nil
}

func (s *sender) onInvalid(token string) {

}

func (s *sender) Close(ctx context.Context) (err error) {
	return
}
