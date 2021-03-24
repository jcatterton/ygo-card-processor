package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"ygo-card-processor/models"
	"ygo-card-processor/pkg/dao"
	"ygo-card-processor/pkg/external"
	"ygo-card-processor/pkg/producer"
	"ygo-card-processor/pkg/reader"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	publicKey  = os.Getenv("PUBLIC_KEY")
	privateKey = os.Getenv("PRIVATE_KEY")
)

func ListenAndServe() error {
	headers := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type"})
	origins := handlers.AllowedOrigins([]string{"*"})
	methods := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS", "DELETE"})

	ctx := context.Background()

	router, err := route(ctx)
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler:      handlers.CORS(headers, origins, methods)(router),
		Addr:         ":8001",
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}
	shutdownGracefully(ctx, server)

	logrus.Info("Starting API server...")
	return server.ListenAndServe()
}

func route(ctx context.Context) (*mux.Router, error) {
	dbClient, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))
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

	fileReader := reader.Reader{}

	p, err := producer.CreateProducer(os.Getenv("BROKER"), os.Getenv("TOPIC"))
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create producer")
		return nil, err
	}

	r := mux.NewRouter()

	r.HandleFunc("/health", checkHealth(&dbHandler)).Methods(http.MethodGet)
	r.HandleFunc("/process", processCards(&dbHandler, &externalRetriever, p)).Methods(http.MethodPost)
	r.HandleFunc("/card/{id}", getCardByNumber(&dbHandler)).Methods(http.MethodGet)
	r.HandleFunc("/card/{id}", addCardById(&dbHandler, &externalRetriever)).Methods(http.MethodPost)
	r.HandleFunc("/card/{id}", updateCard(&dbHandler)).Methods(http.MethodPut)
	r.HandleFunc("/card/{id}", deleteCard(&dbHandler)).Methods(http.MethodDelete)
	r.HandleFunc("/cards", addCardsFromFile(&dbHandler, &externalRetriever, &fileReader)).Methods(http.MethodPost)
	r.HandleFunc("/cards", getCards(&dbHandler)).Methods(http.MethodGet)

	return r, nil
}

func checkHealth(handler dao.DbHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer closeRequestBody(r)
		ctx := r.Context()

		if err := handler.Ping(ctx); err != nil {
			respondWithError(w, http.StatusInternalServerError, "API is running but database ping failed")
			return
		}
		respondWithSuccess(w, http.StatusOK, "API is running and connected to database")
		return
	}
}

func processCards(handler dao.DbHandler, retriever external.ExtRetriever, p producer.KafkaProducer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer closeRequestBody(r)
		ctx := r.Context()

		cardList, err := handler.GetCards(ctx, nil)
		if err != nil {
			logrus.WithError(err).Error("Error getting cards from database")
			respondWithError(w, http.StatusInternalServerError, "Error getting cards from database")
			return
		}

		if err := retriever.RefreshToken(ctx, publicKey, privateKey); err != nil {
			logrus.WithError(err).Error("Error refreshing token")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}

		cardsAdded := 0
		go func() {
			p.Produce("processing_initiated", "card processing has started", false)
			for _, serial := range cardList {
				basicCardInfo, err := retriever.BasicCardSearch(ctx, serial.CardInfo.ExtendedData[0].Value)
				if err != nil {
					logrus.WithError(err).Error("Error performing basic card search")
					p.Produce("processing_error", fmt.Sprintf("error performing basic card search on card with serial '%v'", serial), true)
					continue
				}
				productId := basicCardInfo.Results[0]

				extendedCardInfo, err := retriever.ExtendedCardSearch(ctx, productId)
				if err != nil {
					logrus.WithError(err).Error("Error performing extended card search")
					p.Produce("processing_error", fmt.Sprintf("error performing extended card search on card with serial '%v'", serial), true)
					continue
				}
				cardInfo := extendedCardInfo.Results[0]

				cardPricingInfo, err := retriever.GetCardPricingInfo(ctx, productId)
				if err != nil {
					logrus.WithError(err).Error("Error performing card price search")
					p.Produce("processing_error", fmt.Sprintf("error performing card price search on card with serial '%v'", serial), true)
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

				if _, err := handler.UpdateCardByNumber(ctx, cardInfoWithPrice.CardInfo.ExtendedData[0].Value, cardInfoWithPrice); err != nil {
					logrus.WithError(err).Error("Error updating card")
					p.Produce("processing_error", fmt.Sprintf("error updating card with serial '%v'", serial), true)
					continue
				}

				cardsAdded++
				logrus.Info(fmt.Sprintf("%v out of %v cards processed", cardsAdded, len(cardList)))

				/* One second delay after each card because TCG Player API limits users to 300 API calls per minute.
				With three calls occurring per card, this ensures a maximum of 180 calls per minutes. */
				time.Sleep(1 * time.Second)
			}
			p.Produce("processing_terminated", fmt.Sprintf("card processing has finished - %v cards processed", cardsAdded), false)
		}()

		respondWithSuccess(w, http.StatusOK, fmt.Sprintf("Processing cards has started"))
		return
	}
}

func getCardByNumber(handler dao.DbHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer closeRequestBody(r)
		ctx := r.Context()

		id := mux.Vars(r)["id"]

		result, err := handler.GetCardByNumber(ctx, id)
		if err != nil {
			logrus.WithError(err).Error("Error retrieving card")
			respondWithError(w, http.StatusInternalServerError, "Error retrieving card")
			return
		}

		respondWithSuccess(w, http.StatusOK, result)
		return
	}
}

func addCardsFromFile(handler dao.DbHandler, retriever external.ExtRetriever, fileReader reader.FileReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer closeRequestBody(r)
		ctx := r.Context()

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
			closeRequestBody(r)
			if err = f.Close(); err != nil {
				logrus.WithError(err).Error("Error closing file")
			}
		}()

		cardList, err := fileReader.OpenAndReadFile(f)
		if err != nil {
			logrus.WithError(err).Error("Error reading card list file")
			respondWithError(w, http.StatusInternalServerError, "Error adding cards")
			return
		}

		if err := retriever.RefreshToken(ctx, publicKey, privateKey); err != nil {
			logrus.WithError(err).Error("Error refreshing token")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}

		cardsAdded := 0
		go func() {
			for _, serial := range cardList {
				basicCardInfo, err := retriever.BasicCardSearch(ctx, serial)
				if err != nil {
					logrus.WithError(err).Error("Error performing basic card search")
					continue
				}
				productId := basicCardInfo.Results[0]

				extendedCardInfo, err := retriever.ExtendedCardSearch(ctx, productId)
				if err != nil {
					logrus.WithError(err).Error("Error performing extended card search")
					continue
				}
				cardInfo := extendedCardInfo.Results[0]

				cardPricingInfo, err := retriever.GetCardPricingInfo(ctx, productId)
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

				_, err = handler.AddCard(ctx, cardInfoWithPrice)
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
		}()

		respondWithSuccess(w, http.StatusOK, fmt.Sprintf("Adding cards has started. %v cards are being added.", len(cardList)))
		return
	}
}

func addCardById(handler dao.DbHandler, retriever external.ExtRetriever) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer closeRequestBody(r)
		ctx := r.Context()

		if err := retriever.RefreshToken(ctx, publicKey, privateKey); err != nil {
			logrus.WithError(err).Error("Error refreshing token")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}

		serial := mux.Vars(r)["id"]

		basicCardInfo, err := retriever.BasicCardSearch(ctx, serial)
		if err != nil {
			logrus.WithError(err).Error("Error performing basic card search")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}
		productId := basicCardInfo.Results[0]

		extendedCardInfo, err := retriever.ExtendedCardSearch(ctx, productId)
		if err != nil {
			logrus.WithError(err).Error("Error performing extended card search")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}
		cardInfo := extendedCardInfo.Results[0]

		cardPricingInfo, err := retriever.GetCardPricingInfo(ctx, productId)
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

		result, err := handler.AddCard(ctx, cardInfoWithPrice)
		if err != nil {
			logrus.WithError(err).Error("Error adding card to database")
			respondWithError(w, http.StatusInternalServerError, "Error adding card")
			return
		}

		respondWithSuccess(w, http.StatusOK, result)
		return
	}
}

func updateCard(handler dao.DbHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer closeRequestBody(r)
		ctx := r.Context()

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

		result, err := handler.UpdateCardById(ctx, objectId, card)
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
		defer closeRequestBody(r)
		ctx := r.Context()

		id := mux.Vars(r)["id"]

		err := handler.DeleteCard(ctx, id)
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
		defer closeRequestBody(r)
		ctx := r.Context()

		results, err := handler.GetCards(ctx, nil)
		if err != nil {
			logrus.WithError(err).Error("Error getting cards from database")
			respondWithError(w, http.StatusInternalServerError, "Error getting cards from database")
			return
		}

		respondWithSuccess(w, http.StatusOK, results)
		return
	}
}

func shutdownGracefully(ctx context.Context, server *http.Server) {
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		<-signals

		c, cancel := context.WithTimeout(ctx, 5*time.Second)
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
		return
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
		return
	}
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		logrus.WithError(err).Error("Error encoding response")
	}
}

func closeRequestBody(req *http.Request) {
	if req.Body == nil {
		return
	}
	if err := req.Body.Close(); err != nil {
		logrus.WithError(err).Error("Error closing request body")
		return
	}
	return
}
