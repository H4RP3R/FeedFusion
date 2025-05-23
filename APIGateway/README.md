# API Gateway

**API Gateway** — единая точка входа для клиентов *FeedFusion*. Маршрутизирует запросы к сервисам новостей, комментариев и цензуры, агрегирует данные для составных запросов.

## Эндпоинты

| Метод | Путь         | Описание                               | Параметры                                                                                     |
|-------|--------------|----------------------------------------|-----------------------------------------------------------------------------------------------|
| GET   | /news/latest | Получить последние новости             | page **int** (опциональный),  limit **int** (опциональный)                                    |
| GET   | /news/filter | Поиск новостей по подстроке в названии | contains **string** (обязательный), page **int** (опциональный),  limit **int** (опциональный)|
| GET   | /news/{id}   | Забрать новость с комментариями по UUID| id **UUID**                                                                                   |
| POST  | /comments    | Оставить комментарий (цензура + запись)| **JSON** в теле запроса                                                                       |

## Примеры запросов

### Получить последние новости

```console
GET /news/latest?page=1&limit=10
```

#### Пример ответа

```json
{
    "posts": [
        {
        "id": "0e0f3f31-854f-512d-b4d7-14d341155b20",
        "title": "Заголовок новости",
        "content": "Текст новости (может включать html разметку)",
        "published": "2025-05-22T09:20:28Z",
        "link": "https://source.com/article"
        },
        // ...
    ],
    "pagination": {
        "total_pages": 9,
        "current_page": 1,
        "limit": 10
    }
}
```

### Оставить комментарий к посту

```console
POST /comments
Content-Type: application/json

{
  "post_id": "0e0f3f31-854f-512d-b4d7-14d341155b20",
  "author": "Anna",
  "text": "Some text"
}
```

#### Пример ответа

```json
{
    "id": "2160b2f9-007c-492b-877d-7d3bbb4320e4",
    "post_id": "0e0f3f31-854f-512d-b4d7-14d341155b20",
    "author": "Anna",
    "text": "Some text",
    "published": "2025-05-23T09:49:32.069735211Z"
}
```

#### Ответить на комментарий

```console
POST /comments
Content-Type: application/json

{
  "post_id": "0e0f3f31-854f-512d-b4d7-14d341155b20",
  "parent_id": "2160b2f9-007c-492b-877d-7d3bbb4320e4",
  "author": "Dick",
  "text": "An angry reply!"
}
```

Не прошедший цензуру комментарий возвращает `422 Unprocessable Entity`

### Получить новость по ID

```console
GET /news/0e0f3f31-854f-512d-b4d7-14d341155b20
```

#### Пример ответа

```json
{
    "id": "uuid",
    "title": "string",
    "content": "string",
    "published": "timestamp",
    "link": "string",
    "comments": [
        {
            "id": "uuid",
            "post_id": "uuid",
            "parent_id": "uuid",
            "author": "string",
            "text": "string",
            "published": "timestamp",
            "replies": [...] // вложенные комментарии
        }
    ]
}
```

### Поиск (фильтрация) новостей по подстроке в названии

```console
GET /news/filter?contains=работа&limit=5&page=1
```

## Зависимости

- NewsAggregator
- CommentsService
- CensorshipService
