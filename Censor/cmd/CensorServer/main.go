package main

import (
	"APIGateway/Censor/internal/config"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"github.com/go-chi/render"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

const reqIdStr = "req_id"

// Опции HTTP сервера
const (
	ReadTimeout  = 10 * time.Second
	WriteTimeout = 10 * time.Second
	IdleTimeout  = 60 * time.Second
)

// server - Сервер Censor
type server struct {
	log *slog.Logger
	cfg *config.CensorConfig
}

// Создаём объект сервера.
var srv server

// Comment - структура комментария (урезанная)
type Comment struct {
	Content string `json:"content"`
}

func main() {
	// Инициализируем logger (slog)
	srv.log = slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
	)

	// Инициализируем конфиг (cleanenv)
	srv.cfg = config.MustLoadCensorConfig(srv.log)

	// Переопределяем уровень для логов (slog) значением из конфига
	srv.log = setupLogger(srv.cfg.Env)

	srv.log.Info("starting censor server")
	srv.log.Debug("debug logging enabled")

	srv.log.Debug("load config", "config", srv.cfg)

	// Инициализация роутера (chi)
	srv.log.Debug("init http server")
	router := chi.NewRouter()

	// Middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(httplog.RequestLogger(setupChiLogger()))
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(IdleTimeout))
	router.Use(render.SetContentType(render.ContentTypeJSON))

	// Обработчики запросов
	// Обработчик для цензуры комментариев
	router.Post("/censor", func(w http.ResponseWriter, r *http.Request) {
		const op = "[post]/censor"

		reqId := r.URL.Query().Get(reqIdStr)

		var c Comment
		err := json.NewDecoder(r.Body).Decode(&c)
		if errors.Is(err, io.EOF) {
			srv.log.Debug(op, "empty request", "", reqIdStr, reqId)
			http.Error(w, "empty request", http.StatusBadRequest)
			return
		}
		if err != nil {
			srv.log.Error(op, "decoder", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusInternalServerError)
			return
		}

		srv.log.Info("request body decoded", "content", c.Content, reqIdStr, reqId)

		if censor(c.Content) {
			srv.log.Info(op, "status", "OK", "content", c.Content, reqIdStr, reqId)
			w.WriteHeader(http.StatusOK)
		} else {
			srv.log.Info("StatusBadRequest", "op", op, "content", c.Content, reqIdStr, reqId)
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	// Запуск серверов
	srv.log.Debug("start http server", "address", srv.cfg.Address)
	HTTPSrv := &http.Server{
		Addr:         srv.cfg.Address,
		Handler:      router,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
	}

	if err := HTTPSrv.ListenAndServe(); err != nil {
		srv.log.Error("error starting http server", "error", err.Error())
	}
}

// Переопределяем logger
func setupLogger(env string) *slog.Logger {
	var l *slog.Logger

	switch env {
	case envDev:
		l = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		l = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}
	return l
}

// Chi Logger
func setupChiLogger() *httplog.Logger {
	l := httplog.NewLogger("censor", httplog.Options{
		JSON:             true,
		LogLevel:         slog.LevelInfo,
		Concise:          true,
		RequestHeaders:   true,
		MessageFieldName: "msg",
		TimeFieldName:    "time",
		// TimeFieldFormat: time.RFC850,
		//Tags: map[string]string{
		//	"version": "v1.0-81aa4244d9fc8076a",
		//	"env":     "dev",
		//},
		//QuietDownRoutes: []string{
		//	"/",
		//	"/ping",
		//},
		//QuietDownPeriod: 10 * time.Second,
		// SourceFieldName: "source",
	})

	if srv.cfg.Env == envDev {
		l.Options.LogLevel = slog.LevelDebug
	}

	return l
}

// Функция проверки тела комментария с имеющимся списком слов
func censor(comment string) bool {
	for _, word := range srv.cfg.BadList {
		if strings.Contains(comment, word) {
			return false
		}
	}
	return true
}
