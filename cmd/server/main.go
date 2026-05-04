package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Nakohartum/practicum-metrics/internal/audit"
	fConfig "github.com/Nakohartum/practicum-metrics/internal/config/filestorage"
	mConfig "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	"github.com/Nakohartum/practicum-metrics/internal/handler"
	"github.com/Nakohartum/practicum-metrics/internal/logger"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/Nakohartum/practicum-metrics/internal/service"
)

func main() {
	parseFlags()
	memRepo := setupMemRepo()
	var service service.Service = setupMemService(memRepo)
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	auditor := setupAuditor()
	defer stop()

	if configData.FileWork.fileStoragePath != "" {
		service = setupFileService(memRepo)
	}

	if configData.DatabaseAddress.connectionString != "" {
		service = setupDatabaseService(memRepo)
	}

	server := setupServer(service, auditor, appCtx)
	go func() {
		slog.Info("server started", "addr", configData.Address.String())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("could not listen", "addr", configData.Address.String(), "error", err)
			os.Exit(1)
		}
	}()

	<-appCtx.Done()
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.SaveDataAfterExit(shutdownCtx)

	if err != nil {
		slog.Error("failed to save data after exit", "error", err)
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server shut down")
}

func setupMemRepo() *repository.MemRepo {
	model := models.NewStorageModel()
	config := mConfig.NewMemStorage(model)
	repo := repository.NewMemRepo(config)
	return repo
}

func setupMemService(repo *repository.MemRepo) *service.MetricsService {
	service := service.NewMetricsService(repo, int(configData.FileWork.storeInterval))
	return service
}

func setupDatabaseService(memRepo *repository.MemRepo) *service.DatabaseService {
	dbAdapter := repository.NewPgDatabaseAdapter(configData.DatabaseAddress.connectionString)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := dbAdapter.Open(ctx)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	repo := repository.NewDatabaseRepository(dbAdapter)
	dbService := service.NewDatabaseService(repo, memRepo, int(configData.FileWork.storeInterval))
	return dbService
}

func setupFileService(memRepo *repository.MemRepo) *service.FileService {
	fWriter, err := fConfig.NewFileWriter(configData.FileWork.fileStoragePath)
	if err != nil {
		slog.Error("failed to create file writer", "path", configData.FileWork.fileStoragePath, "error", err)
		os.Exit(1)
		return nil
	}
	fReader, err := fConfig.NewFileReader(configData.FileWork.fileStoragePath)
	if err != nil {
		slog.Error("failed to create file reader", "path", configData.FileWork.fileStoragePath, "error", err)
		os.Exit(1)
		return nil
	}
	fWorker := fConfig.NewFileManager(fReader, fWriter)
	fRepo := repository.NewFileRepo(fWorker)
	fService := service.NewFileService(fRepo, memRepo, int(configData.FileWork.storeInterval))
	return fService
}

func setupRouter(service service.Service) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.StripSlashes)
	router.Use(handler.HashMiddleware(configData.secretKey))
	router.Use(handler.GetZippedDataMiddleware)
	router.Use(handler.GiveZippedDataMiddleware)
	if configData.FileWork.storeInterval == 0 {
		router.Use(handler.SaveAfterPostMiddleware(service))
	}
	return router
}

func setupServer(service service.Service, auditor *audit.Auditor, appCtx context.Context) http.Server {
	router := setupRouter(service)

	metricsHandler := handler.NewMetricsHandler(service, auditor)

	if configData.FileWork.restore {

		res := service.GetAll()
		if len(res) == 0 {
			slog.Info("no data to restore")
		}
		for _, metric := range res {
			switch metric.MType {
			case models.Gauge:
				service.SetData(metric.MType, metric.ID, strconv.FormatFloat(*metric.Value, 'f', -1, 64))
			case models.Counter:
				service.SetData(metric.MType, metric.ID, strconv.FormatInt(*metric.Delta, 10))
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
		r.Post("/updates", logger.AttachLoggingToResponse(metricsHandler.SetMetricsDataHandle()))
	})
	router.Post("/value/", logger.AttachLoggingToResponse(metricsHandler.GetMetricsByNameHandle()))
	if configData.FileWork.storeInterval != 0 {
		go service.RunSaving(appCtx)
	}

	return http.Server{
		Addr:    configData.Address.String(),
		Handler: router,
	}
}

func setupAuditor() *audit.Auditor {
	observers := make([]audit.Observer, 0)

	if configData.AuditFile != "" {
		observers = append(observers, audit.NewFileObserver(configData.AuditFile))
	}

	if configData.AuditUrl != "" {
		observers = append(observers, audit.NewHTTPObserver(configData.AuditUrl, &http.Client{Timeout: 5 * time.Second}))
	}

	return audit.NewAuditor(observers...)
}
