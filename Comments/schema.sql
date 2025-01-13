/*
    Схема БД

// Comment - структура комментария
type Comment struct {
	ID      uint   `json:"id"`       // ID комментария
	NewsId  string `json:"news_id"`  // NewsId - id новости
	Content string `json:"content"`  // Content - текст комментария
	PubTime uint   `json:"pub_time"` // PubTime - время добавления комментария
}
*/

CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    news_id TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,
    pub_time INTEGER NOT NULL DEFAULT (strftime('%s','now')),
CHECK((news_id !='') AND (content !='') AND (pub_time !=0))
);

-- Тестовые данные
INSERT INTO comments (news_id, content) VALUES
    ('Test_News_ID1','Test_News_ID1Content1'),
    ('Test_News_ID1','Test_News_ID1Content2'),
    ('Test_News_ID2','Test_News_ID2Content1'),
    ('Test_News_ID2','Test_News_ID2Content2'),
    ('Test_News_ID2','Test_News_ID2Content3');
