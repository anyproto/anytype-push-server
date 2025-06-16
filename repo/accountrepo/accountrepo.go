//go:generate mockgen -destination mock_accountrepo/mock_accountrepo.go github.com/anyproto/anytype-push-server/repo/accountrepo AccountRepo

package accountrepo

import (
	"context"
	"errors"
	"time"

	"github.com/anyproto/any-sync/app"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/anyproto/anytype-push-server/db"
	"github.com/anyproto/anytype-push-server/domain"
)

const CName = "push.accountrepo"

const collName = "account"

func New() AccountRepo {
	return new(accountRepo)
}

type AccountRepo interface {
	SetAccountTopics(ctx context.Context, accountId string, topics []domain.Topic) error
	GetAccountIdsByTopics(ctx context.Context, topics []domain.Topic) ([]string, error)
	GetTopicsByAccountId(ctx context.Context, accountId string) (topics []domain.Topic, err error)
	app.ComponentRunnable
}

type accountRepo struct {
	coll *mongo.Collection
}

func (r *accountRepo) Init(a *app.App) (err error) {
	r.coll = a.MustComponent(db.CName).(db.Database).Db().Collection(collName)
	return
}

func (r *accountRepo) Name() (name string) {
	return CName
}

func (r *accountRepo) Run(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{"topics", 1}},
	})
	return err
}

func (r *accountRepo) SetAccountTopics(ctx context.Context, accountId string, topics []domain.Topic) error {
	opts := options.Update().SetUpsert(true)
	_, err := r.coll.UpdateByID(
		ctx,
		accountId,
		bson.D{
			{"$set", bson.D{{"topics", topics}, {"updated", time.Now().Unix()}}},
			{"$setOnInsert", bson.D{{"created", time.Now().Unix()}}},
		},
		opts,
	)
	return err
}

type docId struct {
	Id string `bson:"_id"`
}

type withTopics struct {
	Topics []domain.Topic `bson:"topics"`
}

func (r *accountRepo) GetAccountIdsByTopics(ctx context.Context, topics []domain.Topic) ([]string, error) {
	cur, err := r.coll.Find(ctx, bson.M{"topics": bson.M{"$in": topics}}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cur.Close(ctx)
	}()
	var docs []docId
	if err = cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	ids := make([]string, len(docs))
	for i, d := range docs {
		ids[i] = d.Id
	}
	return ids, nil
}

func (r *accountRepo) GetTopicsByAccountId(ctx context.Context, accountId string) (topics []domain.Topic, err error) {
	var topicsRes withTopics
	err = r.coll.FindOne(ctx, bson.M{"_id": accountId}).Decode(&topicsRes)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return topicsRes.Topics, nil
}

func (r *accountRepo) Close(ctx context.Context) error {
	return nil
}
