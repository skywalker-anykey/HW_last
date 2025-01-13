package storage

// Comment - структура комментария
type Comment struct {
	ID      uint   `json:"id"`       // ID комментария
	NewsId  string `json:"news_id"`  // NewsId - id новости
	Content string `json:"content"`  // Content - текст комментария
	PubTime uint   `json:"pub_time"` // PubTime - время добавления комментария
}

// Interface для работы с базой
type Interface interface {
	AddComment(Comment) error
	Comments(string) ([]Comment, error)
}
