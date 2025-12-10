//go:generate mockgen -destination mock_spacerepo/mock_spacerepo.go github.com/anyproto/anytype-push-server/repo/spacerepo SpaceRepo

package spacerepo

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

var (
	ErrSpaceExists   = errors.New("space already exists")
	ErrSpaceNotFound = errors.New("space not found")
)

const CName = "push.spacerepo"

const collName = "space"

func New() SpaceRepo {
	return new(spaceRepo)
}

type SpaceRepo interface {
	Create(ctx context.Context, space domain.Space) (err error)
	Remove(ctx context.Context, space domain.Space) (err error)
	ExistedSpaces(ctx context.Context, spaceIds []string) (existedIds []string, err error)
	app.ComponentRunnable
}

type spaceRepo struct {
	coll *mongo.Collection
}

func (r *spaceRepo) Init(a *app.App) (err error) {
	r.coll = a.MustComponent(db.CName).(db.Database).Db().Collection(collName)
	return
}

func (r *spaceRepo) Name() (name string) {
	return CName
}

func (r *spaceRepo) Run(ctx context.Context) error {
	return nil
}

func (r *spaceRepo) Create(ctx context.Context, space domain.Space) (err error) {
	space.Created = time.Now().Unix()
	_, err = r.coll.InsertOne(ctx, space)
	if mongo.IsDuplicateKeyError(err) {
		err = ErrSpaceExists
	}
	return
}

func (r *spaceRepo) Remove(ctx context.Context, space domain.Space) (err error) {
	res, err := r.coll.DeleteOne(ctx, bson.D{{"_id", space.Id}, {"author", space.Author}})
	if err != nil {
		return
	}
	if res.DeletedCount == 0 {
		return ErrSpaceNotFound
	}
	return
}

type doc struct {
	Id string `bson:"_id"`
}

func (r *spaceRepo) ExistedSpaces(ctx context.Context, spaceIds []string) (existedIds []string, err error) {
	// TODO: we're temporary skip this check because 1-1 spaces not able to register spaces yet
	return spaceIds, nil

	cursor, err := r.coll.Find(
		ctx,
		bson.D{{"_id", bson.D{{"$in", spaceIds}}}},
		options.Find().SetProjection(bson.D{{"_id", 1}}),
	)
	if err != nil {
		return
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()
	var d doc
	for cursor.Next(ctx) {
		if err = cursor.Decode(&d); err != nil {
			return
		}
		existedIds = append(existedIds, d.Id)
	}
	return
}

func (r *spaceRepo) Close(ctx context.Context) error {
	return nil
}
