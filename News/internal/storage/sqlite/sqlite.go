package sqlite

import (
	"APIGateway/News/internal/rss"
	"APIGateway/News/internal/storage"
	"database/sql"
	"errors"
	"fmt"
	"github.com/mattn/go-sqlite3"
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
CREATE TABLE IF NOT EXISTS posts(
	id TEXT NOT NULL UNIQUE,
	title TEXT NOT NULL,
	content TEXT NOT NULL,
	pub_time BIGINT NOT NULL,
	link TEXT NOT NULL,
	CHECK((id !='') AND (title !='') AND (content !='') AND (pub_time !=0) AND (link !=''))
--CREATE INDEX IF NOT EXISTS idx_text ON data(text);
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

// AddPost - добавить новость в БД
func (s *Storage) AddPost(p rss.Post) error {
	const op = "storage.sqlite.AddPost"

	stmt, err := s.db.Prepare("INSERT INTO posts (id, title, content, pub_time, link) VALUES(?,?,?,?,?)")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	_, err = stmt.Exec(&p.ID, &p.Title, &p.Content, &p.PubTime, &p.Link)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) {
			return fmt.Errorf("%s: %w", op, storage.ErrPostExists)
		}

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Posts - вывод определенного количества постов со смещением
func (s *Storage) Posts(skip int, numOnPage int) ([]rss.Post, error) {
	const op = "storage.sqlite.Posts"

	stmt, err := s.db.Prepare("SELECT id,title,content,pub_time,link FROM posts ORDER BY pub_time DESC LIMIT ? OFFSET ?")
	if err != nil {
		return nil, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var posts []rss.Post

	rows, err := stmt.Query(numOnPage, skip)
	if err != nil {
		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	for rows.Next() {
		var post rss.Post
		err = rows.Scan(
			&post.ID,
			&post.Title,
			&post.Content,
			&post.PubTime,
			&post.Link,
		)
		if err != nil {
			return nil, err
		}
		// добавление переменной в массив результатов
		posts = append(posts, post)
	}
	// ВАЖНО не забыть проверить rows.Err()
	return posts, rows.Err()
}

// Count - счетчик количества строк в таблице posts
func (s *Storage) Count() (int, error) {
	const op = "storage.sqlite.Count"

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return count, nil
}

// PostById - получение детальной новости по id
func (s *Storage) PostById(id string) (rss.Post, error) {
	const op = "storage.sqlite.PostById"

	var post rss.Post

	err := s.db.QueryRow("SELECT id,title,content,pub_time,link FROM posts WHERE id = ?", id).Scan(&post.ID, &post.Title, &post.Content, &post.PubTime, &post.Link)
	if err != nil {
		return post, fmt.Errorf("%s: execute statement: %w", op, err)
	}
	if post.ID == "" {
		return post, storage.ErrPostNotFound
	}

	return post, nil
}

// Filter - получение отфильтрованных новостей по заголовку со смещением и ограничением
func (s *Storage) Filter(find string, skip int, numOnPage int) ([]rss.Post, error) {
	const op = "storage.sqlite.Filter"

	find = "%" + find + "%"
	// TODO: поиск без учета регистра не работает с UTF
	stmt, err := s.db.Prepare("SELECT id,title,content,pub_time,link FROM posts WHERE title LIKE ? ORDER BY pub_time DESC LIMIT ? OFFSET ?")
	if err != nil {
		return nil, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var posts []rss.Post

	rows, err := stmt.Query(find, numOnPage, skip)
	if err != nil {
		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	defer func(rows *sql.Rows) {
		_ = rows.Close()
	}(rows)

	for rows.Next() {
		var post rss.Post
		err = rows.Scan(
			&post.ID,
			&post.Title,
			&post.Content,
			&post.PubTime,
			&post.Link,
		)
		if err != nil {
			return nil, err
		}
		// добавление переменной в массив результатов
		posts = append(posts, post)
	}
	// ВАЖНО не забыть проверить rows.Err()
	return posts, rows.Err()
}

// CountOfFilter - счетчик количества новостей по фильтру
func (s *Storage) CountOfFilter(find string) (int, error) {
	const op = "storage.sqlite.CountOfFilter"

	find = "%" + find + "%"
	// TODO: поиск без учета регистра не работает с UTF
	stmt, err := s.db.Prepare("SELECT COUNT(*) FROM posts WHERE title LIKE ?")
	if err != nil {
		return 0, fmt.Errorf("%s: prepare statement: %w", op, err)
	}

	var count int
	err = stmt.QueryRow(find).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return count, nil
}
