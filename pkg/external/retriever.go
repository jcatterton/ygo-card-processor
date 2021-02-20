package external

import "ygo-card-processor/models"

type ExtRetriever interface {
	RefreshToken(publicKey string, privateKey string) error
	BasicCardSearch(serial string) (*models.SearchResponse, error)
	ExtendedCardSearch(productId int) (*models.ExtendedSearchResponse, error)
	GetCardPricingInfo(productId int) (*models.PriceResponse, error)
}
