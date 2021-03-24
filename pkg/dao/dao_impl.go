package dao

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"ygo-card-processor/models"
)

type MongoClient struct {
	Client     *mongo.Client
	Database   string
	Collection string
}

func (db *MongoClient) getCollection() *mongo.Collection {
	return db.Client.Database(db.Database).Collection(db.Collection)
}

func (db *MongoClient) AddCards(ctx context.Context, cardList []interface{}) (int, error) {
	result, err := db.getCollection().InsertMany(ctx, cardList)
	if err != nil {
		return 0, err
	} else if len(result.InsertedIDs) == 0 {
		return 0, errors.New("no cards inserted")
	}
	return len(result.InsertedIDs), nil
}

func (db *MongoClient) AddCard(ctx context.Context, card models.CardWithPriceInfo) (interface{}, error) {
	result, err := db.getCollection().InsertOne(ctx, card)
	if err != nil {
		return 0, err
	}
	return result.InsertedID, nil
}

func (db *MongoClient) UpdateCardById(ctx context.Context, id primitive.ObjectID, card models.CardWithPriceInfo) (*models.CardWithPriceInfo, error) {
	after := options.After
	result := db.getCollection().FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": card}, &options.FindOneAndUpdateOptions{ReturnDocument: &after})
	if result.Err() != nil {
		return nil, result.Err()
	}

	var updatedCard models.CardWithPriceInfo
	if err := result.Decode(&updatedCard); err != nil {
		return nil, err
	}

	return &updatedCard, nil
}

func (db *MongoClient) UpdateCardByNumber(ctx context.Context, serial string, card models.CardWithPriceInfo) (*models.CardWithPriceInfo, error) {
	after := options.After
	result := db.getCollection().FindOneAndUpdate(
		ctx,
		bson.D{
			{"card.extendedData", bson.D{
				{"$elemMatch", bson.D{
					{"value", serial},
				}},
			}},
		},
		bson.M{"$set": card},
		&options.FindOneAndUpdateOptions{ReturnDocument: &after},
	)
	if result.Err() != nil {
		return nil, result.Err()
	}

	var updatedCard models.CardWithPriceInfo
	if err := result.Decode(&updatedCard); err != nil {
		logrus.WithError(err).Error("Error decoding response")
		return nil, err
	}

	return &updatedCard, nil
}

func (db *MongoClient) DeleteCard(ctx context.Context, serial string) error {
	result, err := db.getCollection().DeleteOne(
		ctx,
		bson.D{
			{"card.extendedData", bson.D{
				{"$elemMatch", bson.D{
					{"value", serial},
				}},
			}},
		})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("no cards were deleted")
	}

	return nil
}

func (db *MongoClient) GetCards(ctx context.Context, filters map[string]interface{}) ([]models.CardWithPriceInfo, error) {
	cursor, err := db.getCollection().Find(ctx, filters)
	if err != nil {
		return []models.CardWithPriceInfo{}, err
	}

	var results []models.CardWithPriceInfo
	if err := cursor.All(ctx, &results); err != nil {
		return []models.CardWithPriceInfo{}, err
	}
	return results, nil
}

func (db *MongoClient) GetCardByNumber(ctx context.Context, serial string) (*models.CardWithPriceInfo, error) {
	cursor, err := db.getCollection().Find(
		ctx,
		bson.D{
			{"card.extendedData", bson.D{
				{"$elemMatch", bson.D{
					{"value", serial},
				}},
			}},
		})
	if err != nil {
		return nil, err
	}

	var cards []models.CardWithPriceInfo
	if err := cursor.All(ctx, &cards); err != nil {
		return nil, err
	}

	card := cards[0]
	return &card, nil
}

func (db *MongoClient) Ping(ctx context.Context) error {
	return db.Client.Ping(ctx, readpref.Primary())
}
