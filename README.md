# FeedFusion

**FeedFusion** — это система для агрегации, хранения и публикации новостных постов с возможностью комментирования. Проект реализован на микросервисной архитектуре с использованием языка *Go*, использует современные подходы к построению отказоустойчивых распределённых систем и включает следующие ключевые сервисы:

- **API Gateway** — единая точка входа для клиентов, маршрутизирует запросы к внутренним сервисам.
- **News Aggregator** — агрегирует и хранит новостные статьи, реализует поиск и пагинацию.
- **Comments Service** — хранит и обрабатывает комментарии к постам.
- **Censorship Service** — проверяет комментарии на наличие запрещённых слов.
- **Log Keeper** — централизованный сбор и хранение логов в *Elasticsearch*.

## Архитектура

- Сервисы общаются по *HTTP* (*REST*), логи собираются через очередь сообщений (*Kafka*).
- Для хранения данных используются *PostgreSQL* (новости) и *MongoDB* (комментарии).
- Для цензуры используется отдельный сервис с настраиваемым списком запрещённых слов.
- Логи индексируются в *Elasticsearch* и доступны для анализа в *Kibana*.

## Быстрый старт

```console
# Скопировать схему БД для последующей инициализации
mkdir -p NewsAggregator/db_init
cp NewsAggregator/schema.sql NewsAggregator/db_init/schema.sql
# Установить переменную окружения с паролем для Postgres
export POSTGRES_PASSWORD=some_pass
# Поднять контейнеры
docker compose up --build
```

## Микросервисы

| Сервис             | Описание                                | Документация                                                 |
|--------------------|-----------------------------------------|--------------------------------------------------------------|
| API Gateway        | API-шлюз, маршрутизация запросов        | [APIGateway/README.md](./APIGateway/README.md)               |
| News Aggregator    | Агрегация и выдача новостей             | [NewsAggregator/README.md](./NewsAggregator/README.md)       |
| Comments Service   | Работа с комментариями                  | [CommentsService/README.md](./CommentsService/README.md)     |
| Censorship Service | Цензурирование комментариев             | [CensorshipService/README.md](./CensorshipService/README.md) |
| Log Keeper         | Централизованный сбор логов             | [LogKeeper/README.md](./LogKeeper/README.md)                 |
