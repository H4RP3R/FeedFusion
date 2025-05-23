# Comments Service

Сервис для хранения и получения комментариев к новостям.

## Эндпоинты

| Метод | Путь      | Описание                          | Параметры                       |
|-------|-----------|-----------------------------------|---------------------------------|
| POST  | /comments | Создать комментарий               | **JSON** в теле запроса         |
| GET   | /comments | Получить комментарии к посту      | post_id **UUID** (обязательный) |

## Примеры запросов

### Создать комментарий

```console
POST /comments
Content-Type: application/json
X-Request-Id: 22bca7d6-b3e4-44e1-aae4-06cc07973abd

{
  "post_id": "0e0f3f31-854f-512d-b4d7-14d341155b20",
  "author": "Anna",
  "text": "Some text"
}
```

### Создать ответ на комментарий (вложенный комментарий)

```console
POST /comments
Content-Type: application/json
X-Request-Id: 22bca7d6-b3e4-44e1-aae4-06cc07973abd

{
  "post_id": "0e0f3f31-854f-512d-b4d7-14d341155b20",
  "parent_id": "2160b2f9-007c-492b-877d-7d3bbb4320e4",
  "author": "Dick",
  "text": "An angry reply!"
}
```

### Получить комментарии по ID новости

```console
GET /comments?post_id=0e0f3f31-854f-512d-b4d7-14d341155b20
```

## Зависимости

- MongoDB
