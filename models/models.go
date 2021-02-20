package models

type Card struct {
	ProductId    int            `json:"productId" bson:"productId"`
	Name         string         `json:"name" bson:"name"`
	CleanName    string         `json:"cleanName" bson:"cleanName"`
	ImageUrl     string         `json:"imageUrl" bson:"imageUrl"`
	CategoryId   int            `json:"categoryId" bson:"categoryId"`
	GroupId      int            `json:"groupId" bson:"groupId"`
	Url          string         `json:"url" bson:"url"`
	ModifiedOn   string         `json:"modifiedOn" bson:"modifiedOn"`
	ImageCount   int            `json:"imageCount" bson:"imageCount"`
	PresaleInfo  PresaleInfo    `json:"presaleInfo" bson:"presaleInfo"`
	ExtendedData []ExtendedData `json:"extendedData" bson:"extendedData"`
}

type CardWithPriceInfo struct {
	CardInfo  Card           `json:"card" bson:"card"`
	PriceInfo []PriceResults `json:"priceInfo" bson:"priceInfo"`
}

type PresaleInfo struct {
	IsPresale  bool   `json:"isPresale" bson:"isPresale"`
	ReleasedOn string `json:"releasedOn" bson:"releasedOn"`
	Note       string `json:"note" bson:"note"`
}

type ExtendedData struct {
	Name        string `json:"name" bson:"name"`
	DisplayName string `json:"displayName" bson:"displayName"`
	Value       string `json:"value" bson:"value"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token" bson:"access_token"`
	TokenType   string `json:"token_type" bson:"token_type"`
	ExpiresIn   int    `json:"expires_in" bson:"expires_in"`
	UserName    string `json:"userName" bson:"userName"`
	Issued      string `json:".issues" bson:".issued"`
	Expires     string `json:".expires" bson:".expires"`
}

type SearchResponse struct {
	TotalItems int      `json:"totalItems" bson:"totalItems"`
	Success    bool     `json:"success" bson:"success"`
	Errors     []string `json:"errors" bson:"errors"`
	Results    []int    `json:"results" bson:"results"`
}

type PriceResponse struct {
	Success bool           `json:"success" bson:"success"`
	Errors  []string       `json:"errors" bson:"errors"`
	Results []PriceResults `json:"results" bson:"results"`
}

type PriceResults struct {
	ProductId      int     `json:"productId" bson:"productId"`
	LowPrice       float64 `json:"lowPrice" bson:"lowPrice"`
	MidPrice       float64 `json:"midPrice" bson:"midPrice"`
	HighPrice      float64 `json:"highPrice" bson:"highPrice"`
	MarketPrice    float64 `json:"marketPrice" bson:"marketPrice"`
	DirectLowPrice float64 `json:"directLowPrice" bson:"directLowPrice"`
	SubTypeName    string  `json:"subTypeName" bson:"subTypeName"`
}

type ExtendedSearchResponse struct {
	Success bool     `json:"success" bson:"success"`
	Errors  []string `json:"errors" bson:"errors"`
	Results []Card   `json:"results" bson:"results"`
}

type CardSearchBody struct {
	Filters []CardSearchFilter `json:"filters" bson:"filters"`
}

type CardSearchFilter struct {
	Name   string   `json:"name" bson:"name"`
	Values []string `json:"values" bson:"values"`
}
