package api

import (
	"bytes"
	"errors"
	"github.com/gorilla/mux"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestApi_AddCardsFromFile_ShouldReturn500IfCannotParseMultipartForm(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	retriever := &mocks.ExtRetriever{}
	fileReader := &mocks.FileReader{}

	req, err := http.NewRequest(http.MethodPost, "/cards", nil)
	require.Nil(t, err)

	req.MultipartForm = nil

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardsFromFile(dbHandler, retriever, fileReader))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_AddCardsFromFile_ShouldReturn500IfCannotFindFileWithKeyInput(t *testing.T) {
	path := "../testhelper/output.xlsx"
	file, err := os.Open(path)
	require.Nil(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatal("Unable to close file")
		}
	}()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("test", filepath.Base(path))
	_, err = io.Copy(part, file)
	require.Nil(t, err)
	require.Nil(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, "/cards", body)
	require.Nil(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	dbHandler := &mocks.DbHandler{}
	retriever := &mocks.ExtRetriever{}
	fileReader := &mocks.FileReader{}

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardsFromFile(dbHandler, retriever, fileReader))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_AddCardsFromFile_ShouldReturn500IfReaderReturnsError(t *testing.T) {
	path := "../testhelper/output.xlsx"
	file, err := os.Open(path)
	require.Nil(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatal("Unable to close file")
		}
	}()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("input", filepath.Base(path))
	_, err = io.Copy(part, file)
	require.Nil(t, err)
	require.Nil(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, "/cards", body)
	require.Nil(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	dbHandler := &mocks.DbHandler{}
	retriever := &mocks.ExtRetriever{}
	fileReader := &mocks.FileReader{}
	fileReader.On("OpenAndReadFile", mock.Anything).Return(nil, errors.New("test"))

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardsFromFile(dbHandler, retriever, fileReader))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_AddCardsFromFile_ShouldReturn500IfRefreshTokenReturnsError(t *testing.T) {
	path := "../testhelper/output.xlsx"
	file, err := os.Open(path)
	require.Nil(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatal("Unable to close file")
		}
	}()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("input", filepath.Base(path))
	_, err = io.Copy(part, file)
	require.Nil(t, err)
	require.Nil(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, "/cards", body)
	require.Nil(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	dbHandler := &mocks.DbHandler{}
	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(errors.New("test"))
	fileReader := &mocks.FileReader{}
	fileReader.On("OpenAndReadFile", mock.Anything).Return([]string{"TEST"}, nil)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardsFromFile(dbHandler, retriever, fileReader))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_AddCardsFromFile_ShouldReturn200ButUpdateNoCardsIfBasicCardSearchFails(t *testing.T) {
	path := "../testhelper/output.xlsx"
	file, err := os.Open(path)
	require.Nil(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatal("Unable to close file")
		}
	}()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("input", filepath.Base(path))
	_, err = io.Copy(part, file)
	require.Nil(t, err)
	require.Nil(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, "/cards", body)
	require.Nil(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	dbHandler := &mocks.DbHandler{}
	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(nil, errors.New("test"))
	fileReader := &mocks.FileReader{}
	fileReader.On("OpenAndReadFile", mock.Anything).Return([]string{"TEST"}, nil)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardsFromFile(dbHandler, retriever, fileReader))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
}

func TestApi_AddCardsFromFile_ShouldReturn200ButUpdateNoCardsIfExtendedCardSearchFails(t *testing.T) {
	path := "../testhelper/output.xlsx"
	file, err := os.Open(path)
	require.Nil(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatal("Unable to close file")
		}
	}()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("input", filepath.Base(path))
	_, err = io.Copy(part, file)
	require.Nil(t, err)
	require.Nil(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, "/cards", body)
	require.Nil(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	dbHandler := &mocks.DbHandler{}
	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(&models.SearchResponse{
		Results: []int{123},
	}, nil)
	retriever.On("ExtendedCardSearch", mock.Anything).Return(nil, errors.New("test"))
	fileReader := &mocks.FileReader{}
	fileReader.On("OpenAndReadFile", mock.Anything).Return([]string{"TEST"}, nil)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardsFromFile(dbHandler, retriever, fileReader))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
}

func TestApi_AddCardsFromFile_ShouldReturn200ButUpdateNoCardsIfPricingSearchFails(t *testing.T) {
	path := "../testhelper/output.xlsx"
	file, err := os.Open(path)
	require.Nil(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatal("Unable to close file")
		}
	}()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("input", filepath.Base(path))
	_, err = io.Copy(part, file)
	require.Nil(t, err)
	require.Nil(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, "/cards", body)
	require.Nil(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	dbHandler := &mocks.DbHandler{}
	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(&models.SearchResponse{
		Results: []int{123},
	}, nil)
	retriever.On("ExtendedCardSearch", mock.Anything).Return(&models.ExtendedSearchResponse{
		Results: []models.Card{{}},
	}, nil)
	retriever.On("GetCardPricingInfo", mock.Anything).Return(nil, errors.New("test"))
	fileReader := &mocks.FileReader{}
	fileReader.On("OpenAndReadFile", mock.Anything).Return([]string{"TEST"}, nil)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardsFromFile(dbHandler, retriever, fileReader))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
}

func TestApi_AddCardsFromFile_ShouldReturn200ButUpdateNoCardsIfAddFails(t *testing.T) {
	path := "../testhelper/output.xlsx"
	file, err := os.Open(path)
	require.Nil(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatal("Unable to close file")
		}
	}()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("input", filepath.Base(path))
	_, err = io.Copy(part, file)
	require.Nil(t, err)
	require.Nil(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, "/cards", body)
	require.Nil(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	dbHandler := &mocks.DbHandler{}
	dbHandler.On("AddCard", mock.Anything, mock.Anything).Return(nil, errors.New("test"))
	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(&models.SearchResponse{
		Results: []int{123},
	}, nil)
	retriever.On("ExtendedCardSearch", mock.Anything).Return(&models.ExtendedSearchResponse{
		Results: []models.Card{{}},
	}, nil)
	retriever.On("GetCardPricingInfo", mock.Anything).Return(&models.PriceResponse{
		Results: []models.PriceResults{{MarketPrice: 3.00}},
	}, nil)
	fileReader := &mocks.FileReader{}
	fileReader.On("OpenAndReadFile", mock.Anything).Return([]string{"TEST"}, nil)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardsFromFile(dbHandler, retriever, fileReader))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
}

func TestApi_AddCardsFromFile_ShouldReturn200AndAddCardIfNoErrors(t *testing.T) {
	path := "../testhelper/output.xlsx"
	file, err := os.Open(path)
	require.Nil(t, err)
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatal("Unable to close file")
		}
	}()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("input", filepath.Base(path))
	_, err = io.Copy(part, file)
	require.Nil(t, err)
	require.Nil(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, "/cards", body)
	require.Nil(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	dbHandler := &mocks.DbHandler{}
	dbHandler.On("AddCard", mock.Anything, mock.Anything).Return(nil, nil)
	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(&models.SearchResponse{
		Results: []int{123},
	}, nil)
	retriever.On("ExtendedCardSearch", mock.Anything).Return(&models.ExtendedSearchResponse{
		Results: []models.Card{{}},
	}, nil)
	retriever.On("GetCardPricingInfo", mock.Anything).Return(&models.PriceResponse{
		Results: []models.PriceResults{{MarketPrice: 3.00}},
	}, nil)
	fileReader := &mocks.FileReader{}
	fileReader.On("OpenAndReadFile", mock.Anything).Return([]string{"TEST"}, nil)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardsFromFile(dbHandler, retriever, fileReader))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
}

func TestApi_AddCardById_ShouldReturn500IfRefreshTokenFails(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(errors.New("test"))

	req, err := http.NewRequest(http.MethodPost, "/card", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardById(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_AddCardById_ShouldReturn500IfBasicSearchFails(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(nil, errors.New("test"))

	req, err := http.NewRequest(http.MethodPost, "/card/test", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardById(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_AddCardById_ShouldReturn500IfExtendedSearchFails(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(&models.SearchResponse{
		Results: []int{123},
	}, nil)
	retriever.On("ExtendedCardSearch", mock.Anything).Return(nil, errors.New("test"))

	req, err := http.NewRequest(http.MethodPost, "/card/test", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardById(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_AddCardById_ShouldReturn500IfPricingSearchFails(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(&models.SearchResponse{
		Results: []int{123},
	}, nil)
	retriever.On("ExtendedCardSearch", mock.Anything).Return(&models.ExtendedSearchResponse{
		Results: []models.Card{{}},
	}, nil)
	retriever.On("GetCardPricingInfo", mock.Anything).Return(nil, errors.New("test"))

	req, err := http.NewRequest(http.MethodPost, "/card/test", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardById(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_AddCardById_ShouldReturn500IfAddFails(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("AddCard", mock.Anything, mock.Anything).Return(nil, errors.New("test"))
	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(&models.SearchResponse{
		Results: []int{123},
	}, nil)
	retriever.On("ExtendedCardSearch", mock.Anything).Return(&models.ExtendedSearchResponse{
		Results: []models.Card{{}},
	}, nil)
	retriever.On("GetCardPricingInfo", mock.Anything).Return(&models.PriceResponse{
		Results: []models.PriceResults{{MarketPrice: 3.00}},
	}, nil)

	req, err := http.NewRequest(http.MethodPost, "/card/test", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardById(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_AddCardById_ShouldReturn200IfNoErrors(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("AddCard", mock.Anything, mock.Anything).Return("success", nil)
	retriever := &mocks.ExtRetriever{}
	retriever.On("RefreshToken", mock.Anything, mock.Anything).Return(nil)
	retriever.On("BasicCardSearch", mock.Anything).Return(&models.SearchResponse{
		Results: []int{123},
	}, nil)
	retriever.On("ExtendedCardSearch", mock.Anything).Return(&models.ExtendedSearchResponse{
		Results: []models.Card{{}},
	}, nil)
	retriever.On("GetCardPricingInfo", mock.Anything).Return(&models.PriceResponse{
		Results: []models.PriceResults{{MarketPrice: 3.00}},
	}, nil)

	req, err := http.NewRequest(http.MethodPost, "/card/test", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(addCardById(dbHandler, retriever))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
}

func TestApi_UpdateCard_ShouldReturn500IfInvalidId(t *testing.T) {
	dbHandler := &mocks.DbHandler{}

	req, err := http.NewRequest(http.MethodPut, "/card/test", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(updateCard(dbHandler))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_UpdateCard_ShouldReturn403IfInvalidBody(t *testing.T) {
	dbHandler := &mocks.DbHandler{}

	req, err := http.NewRequest(http.MethodPut, "/card/5df936d80684b40001b3134a", strings.NewReader("test"))
	require.Nil(t, err)
	req = mux.SetURLVars(req, map[string]string{"id": "5df936d80684b40001b3134a"})

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(updateCard(dbHandler))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 400, recorder.Code)
}

func TestApi_UpdateCard_ShouldReturn500IfUpdateFails(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("UpdateCardById", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("test"))

	req, err := http.NewRequest(http.MethodPut, "/card/5df936d80684b40001b3134a", strings.NewReader("{}"))
	require.Nil(t, err)
	req = mux.SetURLVars(req, map[string]string{"id": "5df936d80684b40001b3134a"})

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(updateCard(dbHandler))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_UpdateCard_ShouldReturn200OnSuccess(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("UpdateCardById", mock.Anything, mock.Anything, mock.Anything).Return(&models.CardWithPriceInfo{}, nil)

	req, err := http.NewRequest(http.MethodPut, "/card/5df936d80684b40001b3134a", strings.NewReader("{}"))
	require.Nil(t, err)
	req = mux.SetURLVars(req, map[string]string{"id": "5df936d80684b40001b3134a"})

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(updateCard(dbHandler))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
}

func TestApi_DeleteCard_ShouldReturn500IfDeleteFails(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("DeleteCard", mock.Anything, mock.Anything).Return(errors.New("test"))

	req, err := http.NewRequest(http.MethodDelete, "/card/test", nil)
	require.Nil(t, err)
	req = mux.SetURLVars(req, map[string]string{"id": "TEST-1234"})

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(deleteCard(dbHandler))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_DeleteCard_ShouldReturn200IfNoErrors(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("DeleteCard", mock.Anything, mock.Anything).Return(nil)

	req, err := http.NewRequest(http.MethodDelete, "/card/test", nil)
	require.Nil(t, err)
	req = mux.SetURLVars(req, map[string]string{"id": "TEST-1234"})

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(deleteCard(dbHandler))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
}

func TestApi_GetCards_ShouldReturn500IfGetFails(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("GetCards", mock.Anything, mock.Anything).Return(nil, errors.New("test"))

	req, err := http.NewRequest(http.MethodGet, "/cards", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(getCards(dbHandler))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 500, recorder.Code)
}

func TestApi_GetCards_ShouldReturn200IfNoErrors(t *testing.T) {
	dbHandler := &mocks.DbHandler{}
	dbHandler.On("GetCards", mock.Anything, mock.Anything).Return([]models.CardWithPriceInfo{{}}, nil)

	req, err := http.NewRequest(http.MethodGet, "/cards", nil)
	require.Nil(t, err)

	recorder := httptest.NewRecorder()
	httpHandler := http.HandlerFunc(getCards(dbHandler))
	httpHandler.ServeHTTP(recorder, req)
	require.Equal(t, 200, recorder.Code)
}
