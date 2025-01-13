package main

import (
	"APIGateway/Comments/internal/config"
	"APIGateway/Comments/internal/storage"
	"APIGateway/Comments/internal/storage/sqlite"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	"os"
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

// server - Сервер Comments
type server struct {
	db  storage.Interface
	log *slog.Logger
	cfg *config.CommentsConfig
}

// Создаём объект сервера.
var srv server

func main() {
	// Инициализируем logger (slog)
	srv.log = slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
	)

	// Инициализируем конфиг (cleanenv)
	srv.cfg = config.MustLoadCommentsConfig(srv.log)

	// Переопределяем уровень для логов (slog) значением из конфига
	srv.log = setupLogger(srv.cfg.Env)

	srv.log.Info("starting Comments server")
	srv.log.Debug("debug logging enabled")

	srv.log.Debug("load config", "config", srv.cfg)

	// Инициализация BD (sqlite3)
	s1, err := sqlite.New(srv.cfg.StoragePath)
	if err != nil {
		srv.log.Error("error initializing sqlite storage", "error", err.Error())
		os.Exit(1)
	}
	_ = s1
	srv.db = s1

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
	// обработчик получения комментариев по ID новости
	router.Get("/comment", func(w http.ResponseWriter, r *http.Request) {
		const op = "[get]/comment"

		reqId := r.URL.Query().Get(reqIdStr)
		newsId := r.URL.Query().Get("news_id")

		srv.log.Debug(op, "news_id", newsId, reqIdStr, reqId)

		comments, err := srv.db.Comments(newsId)
		if err != nil {
			srv.log.Error(op, "by id", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = json.NewEncoder(w).Encode(comments)
		if err != nil {
			srv.log.Error(op, "encode", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
	// Обработчик добавления комментария
	router.Post("/comment", func(w http.ResponseWriter, r *http.Request) {
		const op = "[post]/comment"

		reqId := r.URL.Query().Get(reqIdStr)

		var comm storage.Comment
		err := json.NewDecoder(r.Body).Decode(&comm)
		if err != nil {
			srv.log.Error(op, "decode", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusInternalServerError)
			return
		}

		ok, err := censorTest(comm, reqId)
		if err != nil {
			srv.log.Error(op, "censorTest", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusInternalServerError)
			return
		}

		if !ok {
			srv.log.Debug("комментарий не прошел цензуру", "comm", comm, reqIdStr, reqId)
			http.Error(w, "комментарий не прошел цензуру", http.StatusNotAcceptable)
			return
		}

		err = srv.db.AddComment(comm)
		if err != nil {
			srv.log.Error(op, "add comment", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusInternalServerError)
			return
		}
		srv.log.Info(op, "add comment", comm, reqIdStr, reqId)
		w.WriteHeader(http.StatusOK)
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
	l := httplog.NewLogger("comments", httplog.Options{
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

// Тест на сервере проверки цензуры
func censorTest(comment storage.Comment, rId string) (bool, error) {
	const op = "main.censorTest"

	c, err := json.Marshal(comment)
	if err != nil {
		return false, errors.New(op + ": " + err.Error())
	}

	u := srv.cfg.CensorServer + "/censor?" + reqIdStr + "=" + rId
	censor, err := http.Post(u, "application/json", bytes.NewBuffer(c))
	if err != nil {
		return false, errors.New(op + ": " + err.Error())
	}

	statusOK := censor.StatusCode >= 200 && censor.StatusCode < 300
	if !statusOK {
		return false, nil
	}
	return true, nil
}
