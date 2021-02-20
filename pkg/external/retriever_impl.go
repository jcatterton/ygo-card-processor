package external

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"ygo-card-processor/models"
)

type Retriever struct {
	Url    string
	Client http.Client
	Token  string
}

func (r *Retriever) RefreshToken(publicKey string, privateKey string) error {
	body := fmt.Sprintf(
		"grant_type=client_credentials&client_id=%v&client_secret=%v",
		publicKey, privateKey)

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v/token", r.Url), strings.NewReader(body))
	if err != nil {
		logrus.WithError(err).Error("Error creating request")
		return err
	}

	response, err := r.Client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Error performing request")
		return err
	}

	var tokenResponse models.TokenResponse
	if err := json.NewDecoder(response.Body).Decode(&tokenResponse); err != nil {
		logrus.WithError(err).Error("Error decoding response body")
		return err
	}

	r.Token = tokenResponse.AccessToken
	return nil
}

func (r *Retriever) BasicCardSearch(serial string) (*models.SearchResponse, error) {
	filter := models.CardSearchFilter{
		Name:   "Number",
		Values: []string{serial},
	}
	body := models.CardSearchBody{
		Filters: []models.CardSearchFilter{filter},
	}

	bodyJson, err := json.Marshal(body)
	if err != nil {
		logrus.WithError(err).Error("Error marshalling JSON")
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v/v1.37.0/catalog/categories/2/search", r.Url), bytes.NewBuffer(bodyJson))
	if err != nil {
		logrus.WithError(err).Error("Error creating request")
		return nil, err
	}

	if err := r.addHeaders(req); err != nil {
		logrus.WithError(err).Error("Error adding auth token to request")
		return nil, err
	}

	response, err := r.Client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Error performing request")
		return nil, err
	}

	var searchResponse models.SearchResponse
	if err := json.NewDecoder(response.Body).Decode(&searchResponse); err != nil {
		logrus.WithError(err).Error("Error decoding response body")
		return nil, err
	}

	if len(searchResponse.Errors) > 0 {
		err := errors.New(searchResponse.Errors[0])
		logrus.WithError(err).Error("Error in response")
		return nil, err
	}

	return &searchResponse, nil
}

func (r *Retriever) ExtendedCardSearch(productId int) (*models.ExtendedSearchResponse, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/v1.37.0/catalog/products/%v?getExtendedFields=true", r.Url, productId), nil)
	if err != nil {
		logrus.WithError(err).Error("Error creating request")
		return nil, err
	}

	if err := r.addHeaders(req); err != nil {
		logrus.WithError(err).Error("Error adding auth token to request")
		return nil, err
	}

	response, err := r.Client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Error performing request")
		return nil, err
	}

	var searchResponse models.ExtendedSearchResponse
	if err := json.NewDecoder(response.Body).Decode(&searchResponse); err != nil {
		logrus.WithError(err).Error("Error decoding response body")
		return nil, err
	}

	if len(searchResponse.Errors) > 0 {
		err := errors.New(searchResponse.Errors[0])
		logrus.WithError(err).Error("Error in response")
		return nil, err
	}

	return &searchResponse, nil
}

func (r *Retriever) GetCardPricingInfo(productId int) (*models.PriceResponse, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/v1.37.0/pricing/product/%v", r.Url, productId), nil)
	if err != nil {
		logrus.WithError(err).Error("Error creating request")
		return nil, err
	}

	if err := r.addHeaders(req); err != nil {
		logrus.WithError(err).Error("Error adding auth token to request")
		return nil, err
	}

	response, err := r.Client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("Error performing request")
		return nil, err
	}

	var searchResponse models.PriceResponse
	if err := json.NewDecoder(response.Body).Decode(&searchResponse); err != nil {
		logrus.WithError(err).Error("Error decoding response body")
		return nil, err
	}

	if len(searchResponse.Errors) > 0 {
		err := errors.New(searchResponse.Errors[0])
		logrus.WithError(err).Error("Error in response")
		return nil, err
	}

	return &searchResponse, nil
}

func (r *Retriever) addHeaders(req *http.Request) error {
	if r.Token == "" {
		return errors.New("retriever token is empty")
	}
	authToken := fmt.Sprintf("bearer %v", r.Token)
	req.Header.Add("Authorization", authToken)
	req.Header.Add("Content-Type", "application/json")
	return nil
}
