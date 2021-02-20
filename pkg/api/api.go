package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"os"
	"os/signal"
	"time"
	"ygo-card-processor/models"
	"ygo-card-processor/pkg/dao"
	"ygo-card-processor/pkg/external"
	"ygo-card-processor/pkg/reader"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

const (
	publicKey  = "34286c10-c5a6-4fb2-9d3f-f33219873c7d"
	privateKey = "4c2b1a1f-e279-4d64-a9f7-2b04dd1445ee"
)

func ListenAndServe() error {
	headers := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type"})
	origins := handlers.AllowedOrigins([]string{"*"})
	methods := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS", "DELETE"})

	router, err := route()
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler:      handlers.CORS(headers, origins, methods)(router),
		Addr:         ":8001",
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}
	shutdownGracefully(server)

	logrus.Info("Starting API server...")
	return server.ListenAndServe()
}

func route() (*mux.Router, error) {
	dbClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://192.168.1.15:27017"))
	if err != nil {
		return nil, err
	}

	dbHandler := dao.MongoClient{
		Client:     dbClient,
		Database:   "db",
		Collection: "yugioh",
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	externalRetriever := external.Retriever{
		Url:    "https://api.tcgplayer.com",
		Client: client,
		Token:  "",
	}

	r := mux.NewRouter()

	r.HandleFunc("/health", checkHealth(&dbHandler)).Methods(http.MethodGet)
	r.HandleFunc("/process", processCards(&dbHandler, &externalRetriever)).Methods(http.MethodPost)
	r.HandleFunc("/card/{id}", getCardByNumber(&dbHandler)).Methods(http.MethodGet)
	r.HandleFunc("/card/{id}", addCardById(&dbHandler, &externalRetriever)).Methods(http.MethodPost)
	r.HandleFunc("/card/{id}", updateCard(&dbHandler)).Methods(http.MethodPut)
	r.HandleFunc("/card/{id}", deleteCard(&dbHandler)).Methods(http.MethodDelete)
	r.HandleFunc("/cards", addCardsFromFile(&dbHandler, &externalRetriever)).Methods(http.MethodPost)
	r.HandleFunc("/cards", getCards(&dbHandler)).Methods(http.MethodGet)

	return r, nil
}

func checkHealth(handler dao.DbHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler.Ping(context.Background()); err != nil {
			respondWithError(w, http.StatusInternalServerError, "API is running but database ping failed")
			return
		}
		respondWithSuccess(w, http.StatusOK, "API is running and connected to database")
		return
	}
}

func processCards(handler dao.DbHandler, retriever external.ExtRetriever) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cardList, err := handler.GetCards(context.Background(), nil)
		if err != nil {
			logrus.WithError(err).Error("Error getting cards from database")
			respondWithError(w, http.StatusInternalServerError, "Error getting cards from database")
			return
		}

		if err := retriever.RefreshToken(publicKey, privateKey); err != nil {
			logrus.WithError(err).Error("Error refreshing token")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}

		cardsAdded := 0
		for _, serial := range cardList {
			basicCardInfo, err := retriever.BasicCardSearch(serial.CardInfo.ExtendedData[0].Value)
			if err != nil {
				logrus.WithError(err).Error("Error performing basic card search")
				continue
			}
			productId := basicCardInfo.Results[0]

			extendedCardInfo, err := retriever.ExtendedCardSearch(productId)
			if err != nil {
				logrus.WithError(err).Error("Error performing extended card search")
				continue
			}
			cardInfo := extendedCardInfo.Results[0]

			cardPricingInfo, err := retriever.GetCardPricingInfo(productId)
			if err != nil {
				logrus.WithError(err).Error("Error performing card price search")
				continue
			}

			priceResults := make([]models.PriceResults, 0)
			for i := range cardPricingInfo.Results {
				if cardPricingInfo.Results[i].MarketPrice != 0.0 {
					priceResults = append(priceResults, cardPricingInfo.Results[i])
				}
			}

			cardInfoWithPrice := models.CardWithPriceInfo{
				CardInfo:  cardInfo,
				PriceInfo: priceResults,
			}

			if _, err := handler.UpdateCardByNumber(context.Background(), cardInfoWithPrice.CardInfo.ExtendedData[0].Value, cardInfoWithPrice); err != nil {
				logrus.WithError(err).Error("Error updating card")
				continue
			}

			cardsAdded++
			logrus.Info(fmt.Sprintf("%v out of %v cards processed", cardsAdded, len(cardList)))

			/* One second delay after each card because TCG Player API limits users to 300 API calls per minute.
			With three calls occurring per card, this ensures a maximum of 180 calls per minutes. */
			time.Sleep(1 * time.Second)
		}

		respondWithSuccess(w, http.StatusOK, fmt.Sprintf("%v cards updated", cardsAdded))
		return
	}
}

func getCardByNumber(handler dao.DbHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		result, err := handler.GetCardByNumber(context.Background(), id)
		if err != nil {
			logrus.WithError(err).Error("Error retrieving card")
			respondWithError(w, http.StatusInternalServerError, "Error retrieving card")
			return
		}

		respondWithSuccess(w, http.StatusOK, result)
		return
	}
}

func addCardsFromFile(handler dao.DbHandler, retriever external.ExtRetriever) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			logrus.WithError(err).Error("Error parsing file")
			respondWithError(w, http.StatusInternalServerError, "Error adding cards")
			return
		}
		f, _, err := r.FormFile("input")
		if err != nil {
			logrus.WithError(err).Error("Failed to find file with key 'input'")
			respondWithError(w, http.StatusInternalServerError, "Error adding cards")
			return
		}

		defer func() {
			if err = f.Close(); err != nil {
				logrus.WithError(err).Error("Error closing file")
			}
		}()

		cardList, err := reader.OpenAndReadFile(f)
		if err != nil {
			logrus.WithError(err).Error("Error reading card list file")
			respondWithError(w, http.StatusInternalServerError, "Error adding cards")
			return
		}

		if err := retriever.RefreshToken(publicKey, privateKey); err != nil {
			logrus.WithError(err).Error("Error refreshing token")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}

		cardsAdded := 0
		for _, serial := range cardList {
			basicCardInfo, err := retriever.BasicCardSearch(serial)
			if err != nil {
				logrus.WithError(err).Error("Error performing basic card search")
				continue
			}
			productId := basicCardInfo.Results[0]

			extendedCardInfo, err := retriever.ExtendedCardSearch(productId)
			if err != nil {
				logrus.WithError(err).Error("Error performing extended card search")
				continue
			}
			cardInfo := extendedCardInfo.Results[0]

			cardPricingInfo, err := retriever.GetCardPricingInfo(productId)
			if err != nil {
				logrus.WithError(err).Error("Error performing card price search")
				continue
			}

			priceResults := make([]models.PriceResults, 0)
			for i := range cardPricingInfo.Results {
				if cardPricingInfo.Results[i].MarketPrice != 0.0 {
					priceResults = append(priceResults, cardPricingInfo.Results[i])
				}
			}

			cardInfoWithPrice := models.CardWithPriceInfo{
				CardInfo:  cardInfo,
				PriceInfo: priceResults,
			}

			_, err = handler.AddCard(context.Background(), cardInfoWithPrice)
			if err != nil {
				logrus.WithError(err).Error("Error adding card to database")
				continue
			}
			cardsAdded++
			logrus.Info(fmt.Sprintf("%v out of %v cards added", cardsAdded, len(cardList)))

			/* One second delay after each card because TCG Player API limits users to 300 API calls per minute.
			With three calls occurring per card, this ensures a maximum of 180 calls per minutes. */
			time.Sleep(1 * time.Second)
		}

		respondWithSuccess(w, http.StatusOK, fmt.Sprintf("%v out of %v provided cards added to database", cardsAdded, len(cardList)))
		return
	}
}

func addCardById(handler dao.DbHandler, retriever external.ExtRetriever) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := retriever.RefreshToken(publicKey, privateKey); err != nil {
			logrus.WithError(err).Error("Error refreshing token")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}

		serial := mux.Vars(r)["id"]

		basicCardInfo, err := retriever.BasicCardSearch(serial)
		if err != nil {
			logrus.WithError(err).Error("Error performing basic card search")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}
		productId := basicCardInfo.Results[0]

		extendedCardInfo, err := retriever.ExtendedCardSearch(productId)
		if err != nil {
			logrus.WithError(err).Error("Error performing extended card search")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}
		cardInfo := extendedCardInfo.Results[0]

		cardPricingInfo, err := retriever.GetCardPricingInfo(productId)
		if err != nil {
			logrus.WithError(err).Error("Error performing card price search")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}

		priceResults := make([]models.PriceResults, 0)
		for i := range cardPricingInfo.Results {
			if cardPricingInfo.Results[i].MarketPrice != 0.0 {
				priceResults = append(priceResults, cardPricingInfo.Results[i])
			}
		}

		cardInfoWithPrice := models.CardWithPriceInfo{
			CardInfo:  cardInfo,
			PriceInfo: priceResults,
		}

		result, err := handler.AddCard(context.Background(), cardInfoWithPrice)
		if err != nil {
			logrus.WithError(err).Error("Error adding card to database")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}

		respondWithSuccess(w, http.StatusCreated, result)
		return
	}
}

func updateCard(handler dao.DbHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		objectId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			logrus.WithError(err).Error("Error updating card")
			respondWithError(w, http.StatusInternalServerError, "Error updating card")
			return
		}

		var card models.CardWithPriceInfo
		if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
			logrus.WithError(err).Error("Error decoding request body")
			respondWithError(w, http.StatusBadRequest, "Error updating card")
			return
		}

		result, err := handler.UpdateCardById(context.Background(), objectId, card)
		if err != nil {
			logrus.WithError(err).Error("Error updating card")
			respondWithError(w, http.StatusInternalServerError, "Error updating card")
			return
		}

		respondWithSuccess(w, http.StatusOK, result)
		return
	}
}

func deleteCard(handler dao.DbHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		objectId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			logrus.WithError(err).Error("Error deleting card")
			respondWithError(w, http.StatusInternalServerError, "Error deleting card")
			return
		}

		err = handler.DeleteCard(context.Background(), objectId)
		if err != nil {
			logrus.WithError(err).Error("Error deleting card")
			respondWithError(w, http.StatusInternalServerError, "Error deleting card")
			return
		}

		respondWithSuccess(w, http.StatusOK, "Deleted card")
	}
}

func getCards(handler dao.DbHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			logrus.WithError(err).Error("Error parsing form")
			respondWithError(w, http.StatusBadRequest, "Error getting cards from database")
			return
		}

		filters := map[string]interface{}{}
		for key, value := range r.Form {
			filters[key] = value[0]
		}

		results, err := handler.GetCards(context.Background(), filters)
		if err != nil {
			logrus.WithError(err).Error("Error getting cards from database")
			respondWithError(w, http.StatusInternalServerError, "Error getting cards from database")
			return
		}

		respondWithSuccess(w, http.StatusOK, results)
		return
	}
}

func shutdownGracefully(server *http.Server) {
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		<-signals

		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(c); err != nil {
			logrus.WithError(err).Error("Error shutting down server")
		}

		<-c.Done()
		os.Exit(0)
	}()
}

func respondWithSuccess(w http.ResponseWriter, code int, body interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if body == nil {
		logrus.Error("Body is nil, unable to write response")
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logrus.WithError(err).Error("Error encoding response")
	}
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if message == "" {
		logrus.Error("Body is nil, unable to write response")
	}
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		logrus.WithError(err).Error("Error encoding response")
	}
}
