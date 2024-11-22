package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"gopkg.in/go-playground/validator.v9"
)

// Risk represents risk object
type Risk struct {
	ID          string `json:"id"`
	State       string `json:"state" validate:"required,oneof=open closed accepted investigating"`
	Title       string `json:"title" validate:"required"`
	Description string `json:"description" validate:"required"`
}

// InMemoryStore manages risks in-memory
type InMemoryStore struct {
	mu    sync.Mutex
	risks map[string]Risk
}

var (
	store    = InMemoryStore{risks: make(map[string]Risk)}
	validate = validator.New()
	logger   = logrus.New()
)

// CreateRisk handles POST /v1/risks
func CreateRisk(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		writeError(w, errors.New("invalid content-type, expected application/json"), http.StatusUnsupportedMediaType)
		return
	}

	var newRisk Risk
	if err := json.NewDecoder(r.Body).Decode(&newRisk); err != nil {
		writeError(w, errors.New("invalid JSON payload"), http.StatusBadRequest)
		return
	}

	if err := validate.Struct(newRisk); err != nil {
		writeError(w, err, http.StatusBadRequest)
		return
	}

	newRisk.ID = uuid.New().String()

	store.mu.Lock()
	store.risks[newRisk.ID] = newRisk
	store.mu.Unlock()

	writeJSON(w, newRisk, http.StatusCreated)
	logger.WithFields(logrus.Fields{"id": newRisk.ID, "state": newRisk.State}).Info("Risk created successfully")
}

// GetRisks handles GET /v1/risks
func GetRisks(w http.ResponseWriter, r *http.Request) {
	store.mu.Lock()
	defer store.mu.Unlock()

	var risks []Risk
	for _, risk := range store.risks {
		risks = append(risks, risk)
	}

	writeJSON(w, risks, http.StatusOK)
	logger.Info("All risks retrieved successfully")
}

// GetRiskByID handles GET /v1/risks/{id}
func GetRiskByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	store.mu.Lock()
	risk, exists := store.risks[id]
	store.mu.Unlock()

	if !exists {
		writeError(w, errors.New("risk not found"), http.StatusNotFound)
		return
	}

	writeJSON(w, risk, http.StatusOK)
	logger.WithFields(logrus.Fields{"id": id}).Info("Risk retrieved successfully")
}

// writeJSON sends a JSON response
func writeJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.WithError(err).Error("Failed to write JSON response")
	}
}

// writeError sends an error response in JSON format
func writeError(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	logger.WithFields(logrus.Fields{"status_code": statusCode}).Error(err.Error())
}

func main() {
	// Set up logger
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	// Configuration
	port := "8080"
	if p := os.Getenv("APP_PORT"); p != "" {
		port = p
	}

	router := mux.NewRouter()

	// Register routes
	router.HandleFunc("/v1/risks", GetRisks).Methods(http.MethodGet)
	router.HandleFunc("/v1/risks", CreateRisk).Methods(http.MethodPost)
	router.HandleFunc("/v1/risks/{id}", GetRiskByID).Methods(http.MethodGet)

	// Middleware
	router.Use(loggingMiddleware)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.WithField("port", port).Info("Server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-done
	logger.Info("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

// loggingMiddleware logs all incoming requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.WithFields(logrus.Fields{
			"method":   r.Method,
			"path":     r.URL.Path,
			"duration": time.Since(start),
		}).Info("Request handled")
	})
}
