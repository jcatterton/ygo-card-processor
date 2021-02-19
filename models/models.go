package models

type Card struct {
	Name        string `json:"name" bson:"name"`
	Serial      string `json:"serial" bson:"serial"`
	MarketPrice string `json:"marketPrice" bson:"marketPrice"`
	LowestPrice string `json:"lowestPrice" bson:"lowestPrice"`
}
