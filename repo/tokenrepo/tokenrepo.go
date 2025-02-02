//go:generate mockgen -destination mock_tokenrepo/mock_tokenrepo.go github.com/anyproto/anytype-push-server/repo/tokenrepo TokenRepo

package tokenrepo

import (
	"context"
	"errors"
	"time"

	"github.com/anyproto/any-sync/app"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/anyproto/anytype-push-server/db"
	"github.com/anyproto/anytype-push-server/domain"
)

const CName = "push.toknenrepo"

const collName = "token"

var (
	ErrTokenExists = errors.New("token exists")
)

func New() TokenRepo {
	return new(tokenRepo)
}

type TokenRepo interface {
	AddToken(ctx context.Context, token domain.Token) (err error)
	UpdateTokenStatus(ctx context.Context, tokenId string, status domain.TokenStatus) (err error)
	GetActiveTokensByAccountIds(ctx context.Context, accountIds []string) (token []domain.Token, err error)
	app.ComponentRunnable
}

type tokenRepo struct {
	coll *mongo.Collection
}

func (t *tokenRepo) Init(a *app.App) (err error) {
	t.coll = a.MustComponent(db.CName).(db.Database).Db().Collection(collName)
	return
}

func (t *tokenRepo) Run(ctx context.Context) error {
	_, err := t.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{"accountId", 1}, {"status", 1}},
	})
	return err
}

func (t *tokenRepo) Name() (name string) {
	return CName
}

func (t *tokenRepo) AddToken(ctx context.Context, token domain.Token) (err error) {
	now := time.Now().Unix()
	token.Created = now
	token.Updated = now
	_, err = t.coll.InsertOne(ctx, token)
	if mongo.IsDuplicateKeyError(err) {
		return ErrTokenExists
	}
	return
}

func (t *tokenRepo) UpdateTokenStatus(ctx context.Context, tokenId string, status domain.TokenStatus) (err error) {
	_, err = t.coll.UpdateOne(
		ctx,
		bson.D{{"_id", tokenId}},
		bson.D{{"$set", bson.D{
			{"status", status},
			{"updated", time.Now().Unix()},
		}}})
	return
}

func (t *tokenRepo) GetActiveTokensByAccountIds(ctx context.Context, accountIds []string) (tokens []domain.Token, err error) {
	cur, err := t.coll.Find(ctx, bson.D{
		{"accountIds", bson.D{{"$in", accountIds}}},
		{"status", domain.TokenStatusValid},
	})
	if err != nil {
		return
	}
	defer func() {
		_ = cur.Close(ctx)
	}()
	err = cur.All(ctx, &tokens)
	return
}

func (t *tokenRepo) Close(ctx context.Context) (err error) {
	return nil
}
