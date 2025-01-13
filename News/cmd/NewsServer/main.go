package main

import (
	"APIGateway/News/internal/config"
	"APIGateway/News/internal/rss"
	"APIGateway/News/internal/storage"
	"APIGateway/News/internal/storage/sqlite"
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	"os"
	"strconv"
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

// server - Сервер News
type server struct {
	db  storage.Interface
	log *slog.Logger
	cfg *config.NewsConfig
}

// Создаём объект сервера.
var srv server

// Paginates структура с объектом пагинации и новостями, которые отдаются пользователю
type Paginates struct {
	News []rss.Post
	Pag  Pag
}

// Pag Объект пагинации
type Pag struct {
	N      int // Номер текущей страницы
	Pages  int // Количество страниц
	OnPage int // Количество новостей на странице
}

const OnPage = 15

func main() {
	// Инициализируем logger (slog)
	srv.log = slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
	)

	// Инициализируем конфиг (cleanenv)
	srv.cfg = config.MustLoadNewsConfig(srv.log)

	// Переопределяем уровень для логов (slog) значением из конфига
	srv.log = setupLogger(srv.cfg.Env)

	srv.log.Info("starting news server")
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

	// Инициализация получения новостей
	// Период опроса серверов RSS
	t := time.Minute * time.Duration(srv.cfg.RSS.RequestPeriod)

	// Запуск пайпов для каждого url (раздельные горутины)
	for _, url := range srv.cfg.RSS.URLS {
		pipe := cache(readRSS(url, t))
		go reporter(pipe, srv)
	}

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
	// Обработчик Get-запроса на получение новостей постранично
	router.Get("/news", func(w http.ResponseWriter, r *http.Request) {
		const op = "[get]/news"

		reqId := r.URL.Query().Get(reqIdStr)
		p := r.URL.Query().Get("page")
		if p == "" {
			p = "1"
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			srv.log.Error(op, "strconv", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusBadRequest)
			return
		}
		count, err := srv.db.Count()
		if err != nil {
			srv.log.Error(op, "Count", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusBadRequest)
			return
		}

		start := OnPage * (n - 1)
		pages := (count / OnPage) + 1
		post, err := srv.db.Posts(start, OnPage)
		if err != nil {
			srv.log.Error(op, "Posts", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusBadRequest)
			return
		}
		pag := Pag{
			N:      n,
			Pages:  pages,
			OnPage: OnPage,
		}
		send := Paginates{
			News: post,
			Pag:  pag,
		}
		_ = json.NewEncoder(w).Encode(send)
	})

	// Обработчик Get-запроса на получение новости по id
	router.Get("/news/id", func(w http.ResponseWriter, r *http.Request) {
		const op = "[get]/news/id"

		id := r.URL.Query().Get("id")
		reqId := r.URL.Query().Get(reqIdStr)

		post, err := srv.db.PostById(id)
		if err != nil {
			srv.log.Error(op, "PostById", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(post)
	})

	// Обработчик Get-запроса на получение новостей по искомой строке
	router.Get("/news/filter", func(w http.ResponseWriter, r *http.Request) {
		const op = "[get]/news/filter"

		reqId := r.URL.Query().Get(reqIdStr)
		s := r.URL.Query().Get("page")
		if s == "" {
			s = "1"
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			srv.log.Error(op, "strconv", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusBadRequest)
			return
		}
		find := r.URL.Query().Get("s")
		count, err := srv.db.CountOfFilter(find)
		if err != nil {
			srv.log.Error(op, "CountOfFilter", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusBadRequest)
			return
		}
		start := OnPage * (n - 1)
		post, err := srv.db.Filter(find, start, OnPage)
		pages := (count / OnPage) + 1
		if err != nil {
			srv.log.Error(op, "Filter", err.Error(), reqIdStr, reqId)
			http.Error(w, op+": "+err.Error(), http.StatusBadRequest)
			return
		}

		pag := Pag{
			N:      n,
			Pages:  pages,
			OnPage: OnPage,
		}
		send := Paginates{
			News: post,
			Pag:  pag,
		}
		_ = json.NewEncoder(w).Encode(send)
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
	l := httplog.NewLogger("news", httplog.Options{
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

// Читает новости и отправляет в пайп
func readRSS(url string, timer time.Duration) chan rss.Post {
	out := make(chan rss.Post)

	go func() {
		defer close(out)
		for {
			// Получить список новостей если ошибка закрываем пайп
			arPosts, err := rss.GetRSS(url)
			if err != nil {
				return
			}
			// По одной отправляем новости в пайп
			for _, post := range arPosts {
				out <- post
			}

			// После полной отправки ждем период ожидания
			time.Sleep(timer)
		}
	}()
	return out
}

// Пропускает через себя новость только 1 раз
func cache(input <-chan rss.Post) chan rss.Post {
	output := make(chan rss.Post)
	go func() {
		defer close(output)

		// Создаем список уже обработанных новостей
		cacheMap := make(map[string]bool)

		for {
			select {
			case value, ok := <-input:
				// Если канал закрыт, то данных больше не будет и сигнал к завершению работы рутины
				if !ok {
					// Закрываем следующий канал, чтобы оповестить следующую рутину о завершении
					return
				}
				// Если есть ID в cacheMap, то пропускаем, иначе передаем далее
				if cacheMap[value.ID] {
					continue
				} else {
					cacheMap[value.ID] = true
					output <- value
				}
			}
		}
	}()
	return output
}

// Финишный пайп обработки новостей. Добавление в BD
func reporter(input <-chan rss.Post, s server) {

	go func() {
		for value := range input {
			// добавляем все новости в БД
			err := s.db.AddPost(value)
			if err != nil {
				if errors.Is(err, storage.ErrPostExists) {
					// Новость уже была в BD, теперь она есть и в cache
					continue
				}
				s.log.Error("ошибка добавления новости в БД", "error", err.Error())
			} else {
				s.log.Info("Добавлена новость", "post_id", value.ID, "post_title", value.Title)
			}
		}
	}()
}
