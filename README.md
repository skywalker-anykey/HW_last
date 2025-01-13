# HW_last

## Итоговая работа

Все сервисы имеют файлы настроек YAML. Файлы настроек должны находиться с исполняемым файлом в одной директории, либо необходимо перенастроить путь через переменные окружения.

## Gateway

### Настройки по умолчанию:
- Файл с настройками: Gateway.yaml
- Переменная окружения для переопределения пути: GATEWAY_CONFIG_PATH
- Порт: 8080

### Примеры принимаемых запросов:
- Получение новостей (постранично) (GET):

http://localhost:8080/news?page=1

- Получения отфильтрованных по строке новостей. Поиск производиться в названиях новостей (GET):

http://localhost:8080/news/filter?s=go

- Получение детализированной новости (Новости и комментарии к нему) (GET):

http://localhost:8080/news/id?id=https://habr.com/ru/companies/ruvds/articles/872068/
 
- Создания нового комментария с проверкой по цензуре (POST c JSON):

http://localhost:8080/news/comment

<code>{
"news_id": "https://habr.com/ru/companies/ruvds/articles/872068/",
"content": "Тестовый комментарий йцукен"
}</code>

## News

Используется БД Sqlite3

### Настройки по умолчанию:
- Файл с настройками: News.yaml
- Переменная окружения для переопределения пути: NEWS_CONFIG_PATH
- Порт: 8081

### Примеры принимаемых запросов:
- Получение новостей постранично (GET):

http://localhost:8081/news?page=1

- Получение новости по id (GET):

http://localhost:8081/news/id?id=https://habr.com/ru/companies/ruvds/articles/872068/

- Получение новостей по искомой строке (GET):

http://localhost:8081/news/filter?s=go&page=1

## Comments

Используется БД Sqlite3

### Настройки по умолчанию:
- Файл с настройками: Comments.yaml
- Переменная окружения для переопределения пути: COMMENTS_CONFIG_PATH
- Порт: 8082

### Примеры принимаемых запросов:

- Получение комментариев по ID новости (GET)

http://localhost:8082/comment?news_id=https://habr.com/ru/companies/ruvds/articles/872068/

- Создание нового комментария с проверкой по цензуре (POST c JSON):

http://localhost:8082/comment

<code>{
"news_id": "https://habr.com/ru/companies/ruvds/articles/872068/",
"content": "Тестовый комментарий йцукен"
}</code>

## Censor

### Настройки по умолчанию:
- Файл с настройками: Censor.yaml
- Переменная окружения для переопределения пути: CENSOR_CONFIG_PATH
- Порт: 8083

### Примеры принимаемых запросов:

- Проверить комментарий (POST c JSON):

http://localhost:8083/censor

<code>{
"content": "Тестовый комментарий йцукен"
}</code>
