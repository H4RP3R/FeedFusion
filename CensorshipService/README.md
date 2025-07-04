# Censorship Service

Сервис цензуры проверяет текст комментариев на наличие запрещённых слов.

## Использование

Список запрещенных слов задается через файл `forbidden.json`. Каждая запись «слово» имеет паттерн (регулярное выражение) и список исключений (похожие слова, подходящие под регулярное выражение, но на которые цензор не должен срабатывать).

## Эндпоинты

| Метод | Путь   | Описание              | Параметры запроса |
|-------|--------|-----------------------|-------------------|
| POST  | /check | Проверить комментарий | **JSON** в теле   |

## Пример запроса

```console
POST /check
Content-Type: application/json
X-Request-Id: 22bca7d6-b3e4-44e1-aae4-06cc07973abd

{
  "post_id": "31600144-6bb1-58a2-ab39-ff3f4ba66ae6",
  "author": "Dick",
  "text": "Some text"
}
```

## Ответы

- `200 OK` — комментарий прошёл цензуру
- `422 Unprocessable Entity` — комментарий содержит запрещённые слова
