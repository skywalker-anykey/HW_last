/*
    Схема БД

// Post Публикация, получаемая из RSS.
type Post struct {
	ID      string `json:"guid,omitempty"`        // номер записи
	Title   string `json:"title,omitempty"`       // заголовок публикации
	Content string `json:"description,omitempty"` // содержание публикации
	PubTime int64  `json:"pubDate,omitempty"`     // время публикации
	Link    string `json:"link,omitempty"`        // ссылка на источник
}
*/

DROP TABLE IF EXISTS posts;

CREATE TABLE posts (
                       id TEXT NOT NULL UNIQUE,
                       title TEXT NOT NULL,
                       content TEXT NOT NULL,
                       pub_time BIGINT NOT NULL,
                       link TEXT NOT NULL,
                       CHECK((id !='') AND (title !='') AND (content !='') AND (pub_time !=0) AND (link !=''))
);

-- Тестовые данные
INSERT INTO posts (id, title, content, pub_time, link) VALUES
    ('NewId5','NewTitle5','NewContent5',5,'http://new5.url'),
    ('NewId4','NewTitle4','NewContent4',4,'http://new4.url'),
    ('NewId3','NewTitle3','NewContent3',3,'http://new3.url'),
    ('NewId2','NewTitle2','NewContent2',2,'http://new2.url'),
    ('NewId1','NewTitle1','NewContent1',1,'http://new1.url');
