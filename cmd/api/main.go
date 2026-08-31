// O comando api é o ponto de entrada da API de gestão de clínicas.
//
// Atua como composition root: monta os adapters de persistência, injeta-os
// nos services do core e expõe o adapter HTTP com graceful shutdown.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devictorh/clinics/internal/adapter/httpapi"
	"github.com/devictorh/clinics/internal/adapter/memory"
	"github.com/devictorh/clinics/internal/adapter/pixsim"
	"github.com/devictorh/clinics/internal/core/service"
)

func main() {
	if err := run(); err != nil {
		slog.Error("encerrado com erro", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	clinicRepo := memory.NewClinicRepository()
	dentistRepo := memory.NewDentistRepository()
	paymentRepo := memory.NewPaymentRepository()

	clinicSvc := service.NewClinicService(clinicRepo, dentistRepo)
	dentistSvc := service.NewDentistService(dentistRepo, clinicRepo)
	paymentSvc := service.NewPaymentService(paymentRepo, clinicRepo, dentistRepo, pixsim.NewProvider())

	handler := httpapi.NewRouter(clinicSvc, dentistSvc, paymentSvc, logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("api iniciada", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	logger.Info("encerrando api")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return paymentSvc.Shutdown(shutdownCtx)
}
