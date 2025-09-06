package main

import (
	"errors"
	"eventBooker/internal/config"
	"eventBooker/internal/http-server/handlers/event/confirmBooking"
	"eventBooker/internal/http-server/handlers/event/createBooking"
	"eventBooker/internal/http-server/handlers/event/createEvent"
	"eventBooker/internal/http-server/handlers/event/getAllEvents"
	"eventBooker/internal/http-server/handlers/event/getEventInfo"
	"eventBooker/internal/http-server/middleware/mwlogger"
	"eventBooker/internal/lib/logger/handlers/slogpretty"
	"eventBooker/internal/lib/logger/sl"
	"eventBooker/internal/storage/postgres"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("Starting event booker", slog.String("env", cfg.Env))
	log.Debug("Debug messages are enabled")

	storage, err := postgres.InitDB(&cfg.Database)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(mwlogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	fs := http.FileServer(http.Dir("./static/"))
	router.Handle("/static/*", http.StripPrefix("/static/", fs))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/index.html", http.StatusFound)
	})

	router.Post("/events", createEvent.New(log, storage))
	router.Post("/events/{id}/book", createBooking.New(log, storage))
	router.Post("/events/{id}/confirm", confirmBooking.New(log, storage))
	router.Get("/events/{id}", getEventInfo.New(log, storage))
	router.Get("/events", getAllEvents.New(log, storage))

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT, os.Interrupt)

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err = storage.CancelExpiredBookings(); err != nil {
					log.Error("failed to cancel expired bookings", sl.Err(err))
				}
			case <-stop:
				return
			}
		}
	}()

	go func() {
		if err = srv.ListenAndServe(); err != nil && !errors.Is(http.ErrServerClosed, err) {
			log.Error("failed to start server", sl.Err(err))
			stop <- syscall.SIGTERM
		}
	}()

	sign := <-stop

	log.Info("application stopping", slog.String("signal", sign.String()))

	if err = srv.Shutdown(nil); err != nil {
		log.Error("failed to shutdown server", sl.Err(err))
	}

	log.Info("application stopped")

	if err = storage.Close(); err != nil {
		log.Error("failed to close postgres connection", sl.Err(err))
	}

	log.Info("postgres connection closed")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	h := opts.NewPrettyHandler(os.Stdout)

	return slog.New(h)
}
