package dao

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (db *MongoClient) AddCard(ctx context.Context, card models.Card) (interface{}, error) {
	result, err := db.getCollection().InsertOne(ctx, card)
	if err != nil {
		return 0, err
	}
	return result.InsertedID, nil
}

func (db *MongoClient) UpdateCard(ctx context.Context, id primitive.ObjectID, card models.Card) (*models.Card, error) {
	after := options.After
	result := db.getCollection().FindOneAndUpdate(ctx, bson.M{"_id": id}, bson.M{"$set": card}, &options.FindOneAndUpdateOptions{ReturnDocument: &after})
	if result.Err() != nil {
		return nil, result.Err()
	}

	var updatedCard models.Card
	if err := result.Decode(&updatedCard); err != nil {
		return nil, err
	}

	return &updatedCard, nil
}

func (db *MongoClient) DeleteCard(ctx context.Context, id primitive.ObjectID) error {
	_, err := db.getCollection().DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	return nil
}

func (db *MongoClient) GetCards(ctx context.Context, filters map[string]interface{}) ([]models.Card, error) {
	cursor, err := db.getCollection().Find(ctx, filters)
	if err != nil {
		return []models.Card{}, err
	}

	var results []models.Card
	if err := cursor.All(ctx, &results); err != nil {
		return []models.Card{}, err
	}
	return results, nil
}
