package api

import (
	"context"
	"encoding/json"
	"errors"
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

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
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

	r := mux.NewRouter()

	r.HandleFunc("/health", checkHealth()).Methods(http.MethodGet)
	r.HandleFunc("/process", processCards(&dbHandler)).Methods(http.MethodPost)
	r.HandleFunc("/card", addCard(&dbHandler)).Methods(http.MethodPost)
	r.HandleFunc("/card/{id}", updateCard(&dbHandler)).Methods(http.MethodPut)
	r.HandleFunc("/card/{id}", deleteCard(&dbHandler)).Methods(http.MethodDelete)
	r.HandleFunc("/cards", getCards(&dbHandler)).Methods(http.MethodGet)

	return r, nil
}

func checkHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := respondWithSuccess(w, http.StatusOK, "API is running"); err != nil {
			logrus.WithError(err).Error("Failed to write response")
			return
		}
	}
}

func processCards(handler *dao.MongoClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cardList, err := handler.GetCards(context.Background(), nil)
		if err != nil {
			logrus.WithError(err).Error("Error getting cards from database")
			if err := respondWithError(w, http.StatusInternalServerError, "Error getting cards from database"); err != nil {
				logrus.WithError(err).Error("Error writing response")
			}
			return
		}

		result, err := updateCardValues(cardList)
		if err != nil {
			logrus.WithError(err).Error("Error processing cards")
			if err := respondWithError(w, http.StatusInternalServerError, "Error processing cards"); err != nil {
				logrus.WithError(err).Error("Error writing response")
			}
			return
		}

		if err := respondWithSuccess(w, http.StatusOK, fmt.Sprintf("%v cards inserted", result)); err != nil {
			logrus.WithError(err).Error("Error writing response")
		}
		return
	}
}

func addCard(handler *dao.MongoClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var card models.Card

		if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
			logrus.WithError(err).Error("Error decoding request body")
			if err := respondWithError(w, http.StatusBadRequest, "Error adding card to database"); err != nil {
				logrus.WithError(err).Error("Error writing response")
			}
			return
		}

		result, err := handler.AddCard(context.Background(), card)
		if err != nil {
			logrus.WithError(err).Error("Error adding card to database")
			if err := respondWithError(w, http.StatusInternalServerError, "Error adding card to database"); err != nil {
				logrus.WithError(err).Error("Error writing response")
			}
			return
		}

		if err := respondWithSuccess(w, http.StatusCreated, result); err != nil {
			logrus.WithError(err).Error("Error writing response")
		}
		return
	}
}

func updateCard(handler *dao.MongoClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		objectId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			logrus.WithError(err).Error("Error updating card")
			if err := respondWithError(w, http.StatusInternalServerError, "Error updating card"); err != nil {
				logrus.WithError(err).Error("Error writing response")
			}
			return
		}

		var card models.Card

		if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
			logrus.WithError(err).Error("Error decoding request body")
			if err := respondWithError(w, http.StatusBadRequest, "Error updating card"); err != nil {
				logrus.WithError(err).Error("Error writing response")
			}
			return
		}

		result, err := handler.UpdateCard(context.Background(), objectId, card)
		if err != nil {
			logrus.WithError(err).Error("Error updating card")
			if err := respondWithError(w, http.StatusInternalServerError, "Error updating card"); err != nil {
				logrus.WithError(err).Error("Error writing response")
			}
			return
		}

		if err := respondWithSuccess(w, http.StatusOK, result); err != nil {
			logrus.WithError(err).Error("Error writing response")
		}
		return
	}
}

func deleteCard(handler *dao.MongoClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		objectId, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			logrus.WithError(err).Error("Error deleting card")
			if err := respondWithError(w, http.StatusInternalServerError, "Error deleting card"); err != nil {
				logrus.WithError(err).Error("Error writing response")
			}
			return
		}

		err = handler.DeleteCard(context.Background(), objectId)
		if err != nil {
			logrus.WithError(err).Error("Error deleting card")
			if err := respondWithError(w, http.StatusInternalServerError, "Error deleting card"); err != nil {
				logrus.WithError(err).Error("Error writing response")
			}
			return
		}

		if err := respondWithSuccess(w, http.StatusOK, "Deleted card"); err != nil {
			logrus.WithError(err).Error("Error writing response")
		}
	}
}

func getCards(handler *dao.MongoClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			logrus.WithError(err).Error("Error parsing form")
			if err := respondWithError(w, http.StatusBadRequest, "Error getting cards from database"); err != nil {
				logrus.WithError(err).Error("Error writing response")
			}
			return
		}

		filters := map[string]interface{}{}
		for key, value := range r.Form {
			filters[key] = value[0]
		}

		results, err := handler.GetCards(context.Background(), filters)
		if err != nil {
			logrus.WithError(err).Error("Error getting cards from database")
			if err := respondWithError(w, http.StatusInternalServerError, "Error getting cards from database"); err != nil {
				logrus.WithError(err).Error("Error writing response")
			}
			return
		}

		if err := respondWithSuccess(w, http.StatusOK, results); err != nil {
			logrus.WithError(err).Error("Error writing response")
		}
		return
	}
}

func updateCardValues(cards []models.Card) (int, error) {
	return 0, errors.New("unable to process cards at this time")
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

func respondWithSuccess(w http.ResponseWriter, code int, body interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if body == nil {
		return nil
	}
	return json.NewEncoder(w).Encode(body)
}

func respondWithError(w http.ResponseWriter, code int, message string) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if message == "" {
		return nil
	}
	return json.NewEncoder(w).Encode(map[string]string{"error": message})
}
