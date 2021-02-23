package dao

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"ygo-card-processor/models"
)

type DbHandler interface {
	AddCards(ctx context.Context, cardList []interface{}) (int, error)
	AddCard(ctx context.Context, card models.CardWithPriceInfo) (interface{}, error)
	UpdateCardById(ctx context.Context, id primitive.ObjectID, card models.CardWithPriceInfo) (*models.CardWithPriceInfo, error)
	UpdateCardByNumber(ctx context.Context, serial string, card models.CardWithPriceInfo) (*models.CardWithPriceInfo, error)
	DeleteCard(ctx context.Context, serial string) error
	GetCards(ctx context.Context, filters map[string]interface{}) ([]models.CardWithPriceInfo, error)
	GetCardByNumber(ctx context.Context, serial string) (*models.CardWithPriceInfo, error)
	Ping(ctx context.Context) error
}
