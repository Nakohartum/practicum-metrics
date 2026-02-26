package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	config "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	db "github.com/Nakohartum/practicum-metrics/internal/config/db"
	"github.com/Nakohartum/practicum-metrics/internal/handler"
	"github.com/Nakohartum/practicum-metrics/internal/logger"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/Nakohartum/practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	parseFlags()
	conf := setupMemStorage()
	dbAdapter := setupDatabaseAdapter(configData.DatabaseAddress.connectionString)
	err := dbAdapter.Open(context.Background())
	defer dbAdapter.Close(context.Background())
	if err != nil {
		log.Println(err)
	}
	dbRepo := setupPostgreSQLConnection(dbAdapter)
	fileService, repo := setupFileService(conf)
	server := setupServer(repo, dbRepo, fileService)

	go func() {
		log.Println("Server started")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", configData.Address.String(), err)
		}
	}()
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()
	
	err = fileService.SaveData()

	if err != nil {
		log.Println(err)
	}

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server shut down")
}


func setupRouter(fileService *service.FileService) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.StripSlashes)
	router.Use(handler.GetZippedDataMiddleware)
	router.Use(handler.GiveZippedDataMiddleware)
	if configData.FileWork.storeInterval == 0 {
		router.Use(handler.SaveAfterPostMiddleware(fileService))
	}
	return router
}

func setupMemStorage() *config.MemStorage {
	model := models.NewStorageModel()
	conf := config.NewMemStorage(model)
	return conf
}

func setupDatabaseAdapter(connectionString string) db.DatabaseAdapter {
	adapter := db.NewPgDatabaseAdapter(connectionString)
	return adapter
}
 
func setupPostgreSQLConnection(adapter db.DatabaseAdapter) *repository.DatabaseRepository {
	databaseRepo := repository.NewDatabaseRepository(adapter)
	return databaseRepo
}

func setupFileService(conf repository.Storage) (*service.FileService, *repository.MemRepo) {
	reader, err := config.NewFileReader(configData.FileWork.fileStoragePath)	
	if err != nil {
		log.Printf("can't open file for restore: %v", err)
	}
	writer, err := config.NewFileWriter(configData.FileWork.fileStoragePath)
	if err != nil {
		log.Printf("cont open file for writing: %v", err)
	}
	fileManager := config.NewFileManager(reader, writer)
	repo := repository.NewMemRepo(conf, fileManager)
	fileService := service.NewFileService(repo, int(configData.FileWork.storeInterval))
	return fileService, repo
}

func setupServer(repo *repository.MemRepo, dbRepo *repository.DatabaseRepository, fileService *service.FileService) http.Server {
	router := setupRouter(fileService)
	
	metricsService := service.NewMetricsService(repo, dbRepo)
	metricsHandler := handler.NewMetricsHandler(metricsService)

	if configData.FileWork.restore {
		
		res, err := fileService.ReadData()
		if err != nil {
			log.Printf("can't read data from file: %v", err)
		}
		for _, metric := range res {
			switch metric.MType{
				case models.Gauge:
					metricsService.SetData(metric.MType, metric.ID, strconv.FormatFloat(*metric.Value, 'f', -1, 64))
				case models.Counter:
					metricsService.SetData(metric.MType, metric.ID, strconv.FormatInt(*metric.Delta, 10))
			}
		}
	}

	router.Route("/", func(r chi.Router) {
		r.Get("/", metricsHandler.ServePage)
		r.Get("/ping", metricsHandler.Ping().ServeHTTP)
		r.Post("/update", logger.AttachLoggingToResponse(metricsHandler.UpdateMetricsDataHandle()))
		r.Post("/update/{metricType}/{metricName}/{metricValue}", logger.AttachLoggingToResponse(metricsHandler.SetMetricDataHandle()))
		r.Route("/value", func(r chi.Router) {
			r.Post("/", logger.AttachLoggingToResponse(metricsHandler.GetMetricsByNameHandle()))
			r.Get("/{metricType}/{metricName}", logger.AttachLoggingToResponse(metricsHandler.GetMetricDataHandle()))
		})
	})

	if configData.FileWork.storeInterval != 0 {
		go fileService.RunSaving()
	}

	return http.Server{
		Addr:    configData.Address.String(),
		Handler: router,
	}
}