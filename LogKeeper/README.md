# LogK eeper

**Log Keeper** — сервис для централизованного сбора логов из микросервисов через *Kafka* и индексирования логов в *Elasticsearch*.

## Пример лог-сообщения

```json
{
  "timestamp": "2025-05-11T15:20:33Z",
  "ip": "142.250.181.238",
  "status_code": 201,
  "request_id": "31600144-6bb1-58a2-ab39-ff3f4ba66ae6",
  "method": "POST",
  "path": "/comments",
  "duration_sec": 0.025,
  "service": "APIGateway"
}
```

## Зависимости

- Kafka
- Elasticsearch
