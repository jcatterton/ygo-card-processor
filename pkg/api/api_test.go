package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"ygo-card-processor/models"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ygo-card-processor/pkg/testhelper/mocks"
)

func TestApi_CheckHealth_ShouldReturn500IfHandlerReturnsError(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("Ping", mock.Anything).Return(errors.New("test"))

	req, err := http.NewRequest(http.MethodGet, "/health", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(checkHealth(dbHandler))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_CheckHealth_ShouldReturn200IfHandlerReturnsNoError(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("Ping", mock.Anything).Return(nil)

	req, err := http.NewRequest(http.MethodGet, "/health", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(checkHealth(dbHandler))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
}

func TestApi_ProcessCards_ShouldReturn500IfUnableToRetrieveCardsFromDatabase(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("GetCards", mock.Anything, mock.Anything).Return(nil, errors.New("test"))

	retriever := &mocks.ExtRetriever{}

	req, err := http.NewRequest(http.MethodPost, "/process", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(processCards(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_ProcessCards_ShouldReturn500IfUnableToRefreshToken(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("GetCards", mock.Anything, mock.Anything).Return([]models.CardWithPriceInfo{}, nil)

	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(errors.New("test"))

	req, err := http.NewRequest(http.MethodPost, "/process", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(processCards(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_ProcessCards_ShouldReturn200ButNotAddCardIfBasicCardSearchFails(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("GetCards", mock.Anything, mock.Anything).Return([]models.CardWithPriceInfo{
		{CardInfo: models.Card{ExtendedData: []models.ExtendedData{{Value: "test"}}}, PriceInfo: []models.PriceResults{}},
	}, nil)

	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(nil, errors.New("test"))

	req, err := http.NewRequest(http.MethodPost, "/process", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(processCards(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	require.Contains(t, recorder.Body.String(), "0 cards updated")
}

func TestApi_ProcessCards_ShouldReturn200ButNotAddCardIfExtendedCardSearchFails(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("GetCards", mock.Anything, mock.Anything).Return([]models.CardWithPriceInfo{
		{CardInfo: models.Card{ExtendedData: []models.ExtendedData{{Value: "test"}}}, PriceInfo: []models.PriceResults{}},
	}, nil)

	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(&models.SearchResponse{
		Results: []int{123},
	}, nil)
	retriever.On("ExtendedCardSearch", mock.Anything).Return(nil, errors.New("test"))

	req, err := http.NewRequest(http.MethodPost, "/process", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(processCards(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	require.Contains(t, recorder.Body.String(), "0 cards updated")
}

func TestApi_ProcessCards_ShouldReturn200ButNotAddCardIfPricingSearchFails(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("GetCards", mock.Anything, mock.Anything).Return([]models.CardWithPriceInfo{
		{CardInfo: models.Card{ExtendedData: []models.ExtendedData{{Value: "test"}}}, PriceInfo: []models.PriceResults{}},
	}, nil)

	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(&models.SearchResponse{
		Results: []int{123},
	}, nil)
	retriever.On("ExtendedCardSearch", mock.Anything).Return(&models.ExtendedSearchResponse{
		Results: []models.Card{{}},
	}, nil)
	retriever.On("GetCardPricingInfo", mock.Anything).Return(nil, errors.New("test"))

	req, err := http.NewRequest(http.MethodPost, "/process", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(processCards(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	require.Contains(t, recorder.Body.String(), "0 cards updated")
}

func TestApi_ProcessCards_ShouldReturn200ButNotAddCardIfUpdateFails(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("GetCards", mock.Anything, mock.Anything).Return([]models.CardWithPriceInfo{
		{CardInfo: models.Card{ExtendedData: []models.ExtendedData{{Value: "test"}}}, PriceInfo: []models.PriceResults{}},
	}, nil)
	dbHandler.On("UpdateCardByNumber", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("test"))

	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(&models.SearchResponse{
		Results: []int{123},
	}, nil)
	retriever.On("ExtendedCardSearch", mock.Anything).Return(&models.ExtendedSearchResponse{
		Results: []models.Card{{ExtendedData: []models.ExtendedData{{Value: "test"}}}},
	}, nil)
	retriever.On("GetCardPricingInfo", mock.Anything).Return(&models.PriceResponse{
		Results: []models.PriceResults{{MarketPrice: 3.00}},
	}, nil)

	req, err := http.NewRequest(http.MethodPost, "/process", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(processCards(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	require.Contains(t, recorder.Body.String(), "0 cards updated")
}

func TestApi_ProcessCards_ShouldReturn200AndAddCard(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("GetCards", mock.Anything, mock.Anything).Return([]models.CardWithPriceInfo{
		{CardInfo: models.Card{ExtendedData: []models.ExtendedData{{Value: "test"}}}, PriceInfo: []models.PriceResults{}},
	}, nil)
	dbHandler.On("UpdateCardByNumber", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)

	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(&models.SearchResponse{
		Results: []int{123},
	}, nil)
	retriever.On("ExtendedCardSearch", mock.Anything).Return(&models.ExtendedSearchResponse{
		Results: []models.Card{{ExtendedData: []models.ExtendedData{{Value: "test"}}}},
	}, nil)
	retriever.On("GetCardPricingInfo", mock.Anything).Return(&models.PriceResponse{
		Results: []models.PriceResults{{MarketPrice: 3.00}},
	}, nil)

	req, err := http.NewRequest(http.MethodPost, "/process", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(processCards(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
	require.Contains(t, recorder.Body.String(), "1 cards updated")
}

func TestApi_GetCardByNumber_ShouldReturn500IfHandlerReturnsError(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("GetCardByNumber", mock.Anything, mock.Anything).Return(nil, errors.New("test"))

	req, err := http.NewRequest(http.MethodGet, "/card/test", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(getCardByNumber(dbHandler))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_GetCardByNumber_ShouldReturn200IfHandlerReturnsNoError(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("GetCardByNumber", mock.Anything, mock.Anything).Return(&models.CardWithPriceInfo{}, nil)

	req, err := http.NewRequest(http.MethodGet, "/card/test", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(getCardByNumber(dbHandler))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
}
