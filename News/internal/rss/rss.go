package rss

import (
	"github.com/mmcdole/gofeed"
)

// Post Публикация, получаемая из RSS.
type Post struct {
	ID      string `json:"ID,omitempty"`      // Номер записи
	Title   string `json:"Title,omitempty"`   // Заголовок публикации
	Content string `json:"Content,omitempty"` // Содержание публикации
	PubTime int64  `json:"PubTime,omitempty"` // Время публикации
	Link    string `json:"Link,omitempty"`    // Ссылка на источник
}

// GetRSS - получает список новостей из RSS и декодирует в объекты Post
func GetRSS(url string) ([]Post, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, err
	}

	var posts []Post
	if len(feed.Items) == 0 {
		return posts, nil // отсутствие RSS не ошибка, новостей может и не быть
	}

	for _, item := range feed.Items {
		posts = append(posts, Post{
			ID:      item.GUID,
			Title:   item.Title,
			Content: item.Description,
			PubTime: item.PublishedParsed.Unix(),
			Link:    item.Link,
		})
	}
	return posts, nil
}
