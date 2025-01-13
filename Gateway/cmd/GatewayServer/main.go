package main

import (
	"APIGateway/Gateway/internal/config"
	"bytes"
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
	"sync"
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

// server - Сервер Gateway
type server struct {
	log *slog.Logger
	cfg *config.GatewayConfig
}

// Создаём объект сервера.
var srv server

type NewsFullDetailed struct {
	News     News      `json:"news,omitempty"`
	Comments []Comment `json:"comments,omitempty"`
}

type News struct {
	ID      string `json:"ID,omitempty"`      // ID публикации
	Title   string `json:"Title,omitempty"`   // Title Заголовок публикации
	Content string `json:"Content,omitempty"` // Content Содержание публикации
	PubTime uint   `json:"PubTime,omitempty"` // PubTime Время публикации
	Link    string `json:"Link,omitempty"`    // Link Ссылка на источник
}

type Comment struct {
	ID      uint   `json:"id,omitempty"`       // ID комментария
	NewsId  string `json:"news_id,omitempty"`  // NewsId - id новости
	Content string `json:"content,omitempty"`  // Content - текст комментария
	PubTime uint   `json:"pub_time,omitempty"` // PubTime - время добавления комментария
}

func main() {
	// Инициализируем logger (slog)
	srv.log = slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
	)

	// Инициализируем конфиг (cleanenv)
	srv.cfg = config.MustLoadGatewayConfig(srv.log)

	// Переопределяем уровень для логов (slog) значением из конфига
	srv.log = setupLogger(srv.cfg.Env)

	srv.log.Info("starting gateway server")
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
	// Получение новостей на странице
	router.Get("/news", func(w http.ResponseWriter, r *http.Request) {
		reqIdString := middleware.GetReqID(r.Context())
		srv.log.Debug("get news request", reqIdStr, reqIdString)

		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}

		urlStr := srv.cfg.NewsServer + "/news?page=" + page + "&" + reqIdStr + "=" + reqIdString
		resp, err := http.Get(urlStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		//response, _ := json.Marshal(interface{})
		//w.Header().Set("Content-Type", "application/json")
		//w.WriteHeader(http.StatusOK)
		//w.Write(response)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})

	// Обработчик получения отфильтрованных по строке новостей
	router.Get("/news/filter", func(w http.ResponseWriter, r *http.Request) {
		reqIdString := middleware.GetReqID(r.Context())
		srv.log.Debug("get news filter request", reqIdStr, reqIdString)

		page := r.URL.Query().Get("page")

		if page == "" {
			page = "1"
		}

		str := r.URL.Query().Get("s")
		urlStr := srv.cfg.NewsServer + "/news/filter?s=" + str + "&page=" + page + "&" + reqIdStr + "=" + reqIdString
		resp, err := http.Get(urlStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})

	// Получение детализированной новости (Новости и комментарии к нему)
	router.Get("/news/id", func(w http.ResponseWriter, r *http.Request) {
		reqIdString := middleware.GetReqID(r.Context())
		srv.log.Debug("get /news/id", reqIdStr, reqIdString)

		idNews := r.URL.Query().Get("id")
		if idNews == "" {
			srv.log.Error("get /news/id", "id is empty")
			http.Error(w, "id is empty", http.StatusBadRequest)
			return
		}

		var wg sync.WaitGroup
		wg.Add(2)

		var newsFull NewsFullDetailed
		errChanNews := make(chan error)
		errChanComments := make(chan error)

		go getNews(&newsFull, &wg, idNews, reqIdString, errChanNews)
		go getComments(&newsFull, &wg, idNews, reqIdString, errChanComments)

		errNews := <-errChanNews
		errComments := <-errChanComments
		wg.Wait()
		close(errChanNews)
		close(errChanComments)

		if errNews != nil {
			srv.log.Error("get /news/id", "errNews", errNews)
			http.Error(w, errNews.Error(), http.StatusBadRequest)
			return
		}

		if errComments != nil {
			srv.log.Error("get /news/id", "errComments", errComments)
			http.Error(w, errComments.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(newsFull)
	})

	// Обработчик создания нового комментария с проверкой по цензуре
	router.Post("/news/comment", func(w http.ResponseWriter, r *http.Request) {
		reqIdString := middleware.GetReqID(r.Context())
		srv.log.Debug("post news comment", reqIdStr, reqIdString)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		req, err := http.NewRequest("POST", srv.cfg.CommentsServer+"/comment?"+reqIdStr+"="+reqIdString, bytes.NewBuffer(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		defer func(Body io.ReadCloser) {
			_ = Body.Close()
		}(resp.Body)

		if resp.StatusCode == http.StatusOK {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(resp.StatusCode)
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
	l := httplog.NewLogger("gateway", httplog.Options{
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

// Получение новости по id новости
func getNews(news *NewsFullDetailed, wg *sync.WaitGroup, id string, reqIdString string, errChan chan<- error) {
	const op = "main.getNews"
	defer wg.Done()

	urlStr := srv.cfg.NewsServer + "/news/id?id=" + id + "&" + reqIdStr + "=" + reqIdString
	resp, err := http.Get(urlStr)
	if err != nil {
		errChan <- errors.New(op + "[1]: " + err.Error())
		return
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		errChan <- errors.New(resp.Status)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&news.News)
	if err != nil {
		errChan <- errors.New(op + "[2]: " + err.Error())
		return
	}

	srv.log.Debug("getNews return", "news", &news.News)
	errChan <- nil
}

// Получение комментариев по id новости
func getComments(news *NewsFullDetailed, wg *sync.WaitGroup, id string, reqIdString string, errChan chan<- error) {
	const op = "main.getComments"
	defer wg.Done()

	urlStr := srv.cfg.CommentsServer + "/comment?news_id=" + id + "&" + reqIdStr + "=" + reqIdString
	resp, err := http.Get(urlStr)
	if err != nil {
		errChan <- errors.New(op + "[1]: " + err.Error())
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&news.Comments)
	if err != nil {
		errChan <- errors.New(op + "[2]: " + err.Error())
		return
	}
	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		errChan <- errors.New(resp.Status)
		return
	}

	srv.log.Debug("getComments return", "comments", &news.Comments)
	errChan <- nil
}
