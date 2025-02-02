package push

import (
	"context"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/util/crypto"
	"github.com/mr-tron/base58"

	"github.com/anyproto/anytype-push-server/domain"
	"github.com/anyproto/anytype-push-server/pushclient/pushapi"
	"github.com/anyproto/anytype-push-server/queue"
	"github.com/anyproto/anytype-push-server/repo/accountrepo"
	"github.com/anyproto/anytype-push-server/repo/tokenrepo"
)

const CName = "push"

func New() Push {
	return new(push)
}

type Push interface {
	app.Component
}

type push struct {
	tokenRepo   tokenrepo.TokenRepo
	accountRepo accountrepo.AccountRepo
	queue       queue.Queue
	handler     *handler
}

func (p *push) Init(a *app.App) (err error) {
	p.tokenRepo = a.MustComponent(tokenrepo.CName).(tokenrepo.TokenRepo)
	p.accountRepo = a.MustComponent(accountrepo.CName).(accountrepo.AccountRepo)
	p.queue = a.MustComponent(queue.CName).(queue.Queue)
	p.handler = &handler{p: p}
	return pushapi.DRPCRegisterPush(a.MustComponent(server.CName).(server.DRPCServer), p.handler)
}

func (p *push) Name() (name string) {
	return CName
}

func (p *push) AddToken(ctx context.Context, req *pushapi.SetTokenRequest) error {
	if err := checkSignature(req.AccountId, []byte(req.Token), req.Signature); err != nil {
		return err
	}
	peerId, err := peer.CtxPeerId(ctx)
	if err != nil {
		return err
	}
	return p.tokenRepo.AddToken(ctx, domain.Token{
		Id:        req.Token,
		AccountId: req.AccountId,
		PeerId:    peerId,
		Platform:  domain.Platform(req.Platform),
		Status:    domain.TokenStatusValid,
	})
}

func (p *push) SubscribeAll(ctx context.Context, req *pushapi.SubscribeAllRequest) error {
	if err := checkSignature(req.AccountId, req.Payload, req.Signature); err != nil {
		return err
	}
	var rawTopics = &pushapi.Topics{}
	if err := rawTopics.Unmarshal(req.Payload); err != nil {
		return err
	}
	topics, err := convertTopics(rawTopics)
	if err != nil {
		return err
	}
	return p.accountRepo.SetAccountTopics(ctx, req.AccountId, topics)
}

func (p *push) Notify(ctx context.Context, req *pushapi.NotifyRequest) error {
	if err := checkSignature(req.AccountId, req.Payload, req.Signature); err != nil {
		return err
	}
	var rawTopics = &pushapi.Topics{}
	if err := rawTopics.Unmarshal(req.Payload); err != nil {
		return err
	}
	topics, err := convertTopics(rawTopics)
	if err != nil {
		return err
	}
	message := queue.Message{
		IgnoreAccountId: req.AccountId,
		Topics:          topics,
	}
	return p.queue.Add(ctx, message)
}

func checkSignature(accountId string, payload, signature []byte) (err error) {
	pubKey, err := crypto.DecodeAccountAddress(accountId)
	if err != nil {
		return err
	}
	valid, err := pubKey.Verify(payload, signature)
	if err != nil {
		return err
	}
	if !valid {
		return pushapi.ErrInvalidSignature
	}
	return nil
}

func convertTopics(topics *pushapi.Topics) (result []domain.Topic, err error) {
	result = make([]domain.Topic, len(topics.Topics))

	var pks = map[string]crypto.PubKey{}

	getKey := func(spaceKey []byte) (crypto.PubKey, error) {
		if key, ok := pks[string(spaceKey)]; ok {
			return key, nil
		}
		key, decErr := crypto.UnmarshalEd25519PublicKey(spaceKey)
		if decErr != nil {
			return nil, decErr
		}
		pks[string(spaceKey)] = key
		return key, nil
	}

	var (
		key   crypto.PubKey
		valid bool
	)
	for i, topic := range topics.Topics {
		if key, err = getKey(topic.SpaceKey); err != nil {
			return nil, err
		}
		if valid, err = key.Verify([]byte(topic.Topic), topic.Signature); err != nil {
			return nil, err
		}
		if !valid {
			return nil, pushapi.ErrInvalidTopicSignature
		}
		result[i] = domain.Topic(base58.Encode(topic.SpaceKey) + "/" + topic.Topic)
	}
	return
}
