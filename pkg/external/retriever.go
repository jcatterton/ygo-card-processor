package external

import (
	"context"

	"ygo-card-processor/models"
)

type ExtRetriever interface {
	RefreshToken(ctx context.Context, publicKey string, privateKey string) error
	BasicCardSearch(ctx context.Context, serial string) (*models.SearchResponse, error)
	ExtendedCardSearch(ctx context.Context, productId int) (*models.ExtendedSearchResponse, error)
	GetCardPricingInfo(ctx context.Context, productId int) (*models.PriceResponse, error)
}
