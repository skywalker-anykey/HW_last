package sqlite

import (
	"APIGateway/Comments/internal/storage"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

// New - конструктор BD
func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	stmt, err := db.Prepare(`
CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    news_id TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,
    pub_time INTEGER NOT NULL DEFAULT (strftime('%s','now')),
CHECK((news_id !='') AND (content !='') AND (pub_time !=0))
);
`)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

// AddComment - добавление комментария в БД
func (s *Storage) AddComment(comment storage.Comment) error {
	const op = "storage.sqlite.AddComment"

	stmt, err := s.db.Prepare("INSERT INTO comments (news_id,content) VALUES(?,?)")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(&comment.NewsId, &comment.Content)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Comments - получение комментариев по id поста
func (s *Storage) Comments(NewsID string) ([]storage.Comment, error) {
	const op = "storage.sqlite.Comments"

	stmt, err := s.db.Prepare("SELECT comments.id,comments.news_id,comments.content,comments.pub_time FROM comments WHERE comments.news_id = ? ORDER BY comments.pub_time DESC")
	if err != nil {
		return nil, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var comments []storage.Comment

	rows, err := stmt.Query(NewsID)
	if err != nil {
		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	for rows.Next() {
		var comment storage.Comment
		err = rows.Scan(
			&comment.ID,
			&comment.NewsId,
			&comment.Content,
			&comment.PubTime,
		)
		if err != nil {
			return nil, err
		}
		// добавление переменной в массив результатов
		comments = append(comments, comment)
	}
	// ВАЖНО не забыть проверить rows.Err()
	return comments, rows.Err()
}
