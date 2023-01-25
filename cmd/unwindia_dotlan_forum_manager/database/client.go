package database

import (
	"context"
	"github.com/GSH-LAN/Unwindia_dotlan_forum_manager/cmd/unwindia_dotlan_forum_manager/environment"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

const (
	CollectionName = "dotlan_forum_manager"
	DatabaseName   = "unwindia"
	DefaultTimeout = 10 * time.Second
)

// DatabaseClient is the client-interface for the main mongodb database
type DatabaseClient interface {
	// Upsert creates or updates an DotlanForumStatus entry
	Upsert(ctx context.Context, entry *DotlanForumStatus) error
	// Get returns an existing DotlanForumStatus by the given id. Id is the id of the match within dotlan (tcontest.tcid)
	Get(ctx context.Context, id string) (*DotlanForumStatus, error)
	// List returns all existing DotlanForumStatus entries in a Result chan
	List(ctx context.Context, filter interface{}, resultChan chan Result)
}

func NewClient(ctx context.Context, env *environment.Environment) (*DatabaseClientImpl, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(env.MongoDbURI))
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}

	ctx, _ = context.WithTimeout(ctx, 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}

	return NewClientWithDatabase(ctx, client.Database(DatabaseName))
}

func NewClientWithDatabase(ctx context.Context, db *mongo.Database) (*DatabaseClientImpl, error) {

	dbClient := DatabaseClientImpl{
		ctx:        ctx,
		collection: db.Collection(CollectionName),
	}

	return &dbClient, nil
}

type DatabaseClientImpl struct {
	ctx        context.Context
	collection *mongo.Collection
}

func (d DatabaseClientImpl) Upsert(ctx context.Context, entry *DotlanForumStatus) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	filter := bson.D{{"_id", entry.ID}}

	updateResult, err := d.collection.ReplaceOne(ctx, filter, entry, options.Replace().SetUpsert(true))

	log.Debug().Interface("updateResult", *updateResult).Msg("Update result")

	return err
}

func (d DatabaseClientImpl) Get(ctx context.Context, id string) (*DotlanForumStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	filter := bson.D{{"_id", id}}
	result := d.collection.FindOne(ctx, filter)
	if result.Err() != nil {
		return nil, result.Err()
	}

	var entry DotlanForumStatus
	err := result.Decode(&entry)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (d DatabaseClientImpl) List(ctx context.Context, filter interface{}, resultChan chan Result) {
	var listResult []DotlanForumStatus

	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	if filter == nil {
		filter = bson.D{}
	}

	cur, err := d.collection.Find(ctx, filter)

	if err != nil {
		resultChan <- Result{Result: nil, Error: err}
		return
	}

	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var result DotlanForumStatus
		if err := cur.Decode(&result); err != nil {
			log.Error().Err(err).Msg("Error decoding document")
		} else {
			listResult = append(listResult, result)
		}

	}

	resultChan <- Result{Result: listResult, Error: nil}
}
