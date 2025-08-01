package push

import (
	"context"
	"errors"
	"slices"

	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/app/logger"
	"github.com/anyproto/any-sync/metric"
	"github.com/anyproto/any-sync/net/peer"
	"github.com/anyproto/any-sync/net/rpc/server"
	"github.com/anyproto/any-sync/util/crypto"
	"github.com/mr-tron/base58"

	"go.uber.org/zap"

	"github.com/anyproto/anytype-push-server/domain"
	"github.com/anyproto/anytype-push-server/pushclient/pushapi"
	"github.com/anyproto/anytype-push-server/queue"
	"github.com/anyproto/anytype-push-server/repo/accountrepo"
	"github.com/anyproto/anytype-push-server/repo/spacerepo"
	"github.com/anyproto/anytype-push-server/repo/tokenrepo"
)

const CName = "push"

var log = logger.NewNamed(CName)

func New() Push {
	return new(push)
}

type Push interface {
	app.Component
}

type push struct {
	tokenRepo   tokenrepo.TokenRepo
	accountRepo accountrepo.AccountRepo
	spaceRepo   spacerepo.SpaceRepo
	queue       queue.Queue
	metric      metric.Metric
	handler     *handler
}

func (p *push) Init(a *app.App) (err error) {
	p.tokenRepo = a.MustComponent(tokenrepo.CName).(tokenrepo.TokenRepo)
	p.accountRepo = a.MustComponent(accountrepo.CName).(accountrepo.AccountRepo)
	p.spaceRepo = a.MustComponent(spacerepo.CName).(spacerepo.SpaceRepo)
	p.queue = a.MustComponent(queue.CName).(queue.Queue)
	p.metric = a.MustComponent(metric.CName).(metric.Metric)
	p.handler = &handler{p: p}
	return pushapi.DRPCRegisterPush(a.MustComponent(server.CName).(server.DRPCServer), p.handler)
}

func (p *push) Name() (name string) {
	return CName
}

func (p *push) AddToken(ctx context.Context, req *pushapi.SetTokenRequest) error {
	accPubKey, err := peer.CtxPubKey(ctx)
	if err != nil {
		return err
	}
	peerId, err := peer.CtxPeerId(ctx)
	if err != nil {
		return err
	}
	return p.tokenRepo.AddToken(ctx, domain.Token{
		Id:        req.Token,
		AccountId: accPubKey.Account(),
		PeerId:    peerId,
		Platform:  domain.Platform(req.Platform),
		Status:    domain.TokenStatusValid,
	})
}

func (p *push) RevokeToken(ctx context.Context) error {
	accPubKey, err := peer.CtxPubKey(ctx)
	if err != nil {
		return err
	}
	peerId, err := peer.CtxPeerId(ctx)
	if err != nil {
		return err
	}
	return p.tokenRepo.RevokeToken(ctx, accPubKey.Account(), peerId)
}

func (p *push) SubscribeAll(ctx context.Context, req *pushapi.SubscribeAllRequest) error {
	accPubKey, err := peer.CtxPubKey(ctx)
	if err != nil {
		return err
	}
	topics, err := convertTopics(req.Topics)
	if err != nil {
		return err
	}
	return p.accountRepo.SetAccountTopics(ctx, accPubKey.Account(), topics)
}

func (p *push) Notify(ctx context.Context, req *pushapi.NotifyRequest, silent bool) error {
	accPubKey, err := peer.CtxPubKey(ctx)
	if err != nil {
		return err
	}
	topics, err := convertTopics(req.Topics)
	if err != nil {
		return err
	}
	valid, err := accPubKey.Verify(req.Message.Payload, req.Message.Signature)
	if err != nil {
		return err
	}
	if !valid {
		return pushapi.ErrInvalidSignature
	}

	// mak a list of unique spaceKeys
	var spaceKeys = make([]string, 0, len(topics))
	for _, topic := range topics {
		spaceKey := topic.SpaceKeyBase58()
		if !slices.Contains(spaceKeys, spaceKey) {
			spaceKeys = append(spaceKeys, spaceKey)
		}
	}
	validSpaceKeys, err := p.spaceRepo.ExistedSpaces(ctx, spaceKeys)
	if err != nil {
		return err
	}
	// filter by registered space keys
	var filteredTopics = topics[:0]
	for _, topic := range topics {
		if slices.Contains(validSpaceKeys, topic.SpaceKeyBase58()) {
			filteredTopics = append(filteredTopics, topic)
		}
	}
	topics = filteredTopics

	if len(topics) == 0 {
		return nil
	}

	message := queue.Message{
		KeyId:     req.Message.KeyId,
		Payload:   req.Message.Payload,
		Signature: req.Message.Signature,
		GroupId:   req.GroupId,
		Topics:    topics,
		Silent:    silent,
	}
	if !silent {
		message.IgnoreAccountId = accPubKey.Account()
	}
	return p.queue.Add(ctx, message)
}

func (p *push) CreateSpace(ctx context.Context, key []byte, signature []byte) (err error) {
	accPubKey, err := peer.CtxPubKey(ctx)
	if err != nil {
		return err
	}
	if err = checkSpaceSignature(accPubKey.Account(), key, signature); err != nil {
		return
	}
	err = p.spaceRepo.Create(ctx, domain.Space{
		Id:     base58.Encode(key),
		Author: accPubKey.Account(),
	})
	if errors.Is(err, spacerepo.ErrSpaceExists) {
		err = pushapi.ErrSpaceExists
	}
	return nil
}

func (p *push) RemoveSpace(ctx context.Context, key []byte, signature []byte) (err error) {
	accPubKey, err := peer.CtxPubKey(ctx)
	if err != nil {
		return err
	}
	if err = checkSpaceSignature(accPubKey.Account(), key, signature); err != nil {
		return
	}
	err = p.spaceRepo.Remove(ctx, domain.Space{
		Id:     base58.Encode(key),
		Author: accPubKey.Account(),
	})
	if errors.Is(err, spacerepo.ErrSpaceExists) {
		err = pushapi.ErrSpaceExists
	}
	return nil
}

func checkSpaceSignature(identity string, spaceKey, signature []byte) error {
	key, err := crypto.UnmarshalEd25519PublicKey(spaceKey)
	if err != nil {
		return err
	}
	valid, err := key.Verify([]byte(identity), signature)
	if err != nil {
		return err
	}
	if !valid {
		return pushapi.ErrInvalidSignature
	}
	return nil
}

func (p *push) Subscriptions(ctx context.Context) (topics *pushapi.Topics, err error) {
	accPubKey, err := peer.CtxPubKey(ctx)
	if err != nil {
		return nil, err
	}

	dTopics, err := p.accountRepo.GetTopicsByAccountId(ctx, accPubKey.Account())
	if err != nil {
		return nil, err
	}

	topics = &pushapi.Topics{
		Topics: make([]*pushapi.Topic, len(dTopics)),
	}

	for i, dtopic := range dTopics {
		raw, err := dtopic.SpaceKeyRaw()
		if err != nil {
			return nil, err
		}
		topics.Topics[i] = &pushapi.Topic{
			SpaceKey: raw,
			Topic:    dtopic.Topic(),
		}
	}
	return
}

func (p *push) Subscribe(ctx context.Context, topics *pushapi.Topics) error {
	accPubKey, err := peer.CtxPubKey(ctx)
	if err != nil {
		return err
	}

	dTopics, err := convertTopics(topics)
	if err != nil {
		return err
	}
	return p.accountRepo.SetAccountTopics(ctx, accPubKey.Account(), dTopics)

}

func (p *push) Unsubscribe(ctx context.Context, topics *pushapi.Topics) error {
	accPubKey, err := peer.CtxPubKey(ctx)
	if err != nil {
		return err
	}

	dTopics, err := convertTopics(topics)
	if err != nil {
		return err
	}

	if len(dTopics) == 0 {
		return nil
	}

	currentTopics, err := p.accountRepo.GetTopicsByAccountId(ctx, accPubKey.Account())
	if err != nil {
		return err
	}

	removeSet := make(map[domain.Topic]struct{}, len(dTopics))
	for _, t := range dTopics {
		removeSet[t] = struct{}{}
	}

	remainingTopics := make([]domain.Topic, 0, len(currentTopics))
	for _, t := range currentTopics {
		if _, shouldRemove := removeSet[t]; !shouldRemove {
			remainingTopics = append(remainingTopics, t)
		}
	}

	return p.accountRepo.SetAccountTopics(ctx, accPubKey.Account(), remainingTopics)
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
			log.Info("convert topics unmarshall err: ", zap.Int("len", len(spaceKey)), zap.ByteString("spaceKey", spaceKey))
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
		result[i] = domain.NewTopic(topic.SpaceKey, topic.Topic)
	}
	return
}
