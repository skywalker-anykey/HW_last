package storage

import (
	"APIGateway/News/internal/rss"
	"errors"
)

var (
	ErrPostExists   = errors.New("post exists")
	ErrPostNotFound = errors.New("post not found")
)

// Interface для работы с базой
type Interface interface {
	AddPost(rss.Post) error                      // Добавить новость в БД
	Posts(int, int) ([]rss.Post, error)          // Вывод определенного количества постов со смещением
	Count() (int, error)                         // Счетчик количества строк в таблице posts
	PostById(string) (rss.Post, error)           // Получение детальной новости по id
	Filter(string, int, int) ([]rss.Post, error) // Получение отфильтрованных новостей по заголовку со смещением и ограничением
	CountOfFilter(string) (int, error)           // Счетчик количества новостей по фильтру
}
