# Subscription Service

REST API для агрегации данных об онлайн-подписках пользователей.

## Стек технологий

- **Go 1.23**
- **PostgreSQL 16**
- **chi** — HTTP-роутер
- **pgx v5** — драйвер PostgreSQL
- **golang-migrate** — миграции БД
- **swaggo** — Swagger-документация
- **Docker / Docker Compose**

## Запуск

```bash
docker compose up --build
```

Сервис будет доступен по адресу: `http://localhost:8080`

## API

### Swagger UI

После запуска доступен по адресу: `http://localhost:8080/swagger/index.html`

### Эндпоинты

| Метод    | Путь                          | Описание                                      |
|----------|-------------------------------|-----------------------------------------------|
| `POST`   | `/api/v1/subscriptions`       | Создать подписку                               |
| `GET`    | `/api/v1/subscriptions`       | Список подписок (пагинация + фильтры)          |
| `GET`    | `/api/v1/subscriptions/{id}`  | Получить подписку по ID                         |
| `PUT`    | `/api/v1/subscriptions/{id}`  | Обновить подписку                               |
| `DELETE` | `/api/v1/subscriptions/{id}`  | Удалить подписку                                |
| `GET`    | `/api/v1/subscriptions/cost`  | Рассчитать суммарную стоимость за период         |

### Примеры запросов

**Создание подписки:**

```bash
curl -X POST http://localhost:8080/api/v1/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "service_name": "Yandex Plus",
    "price": 400,
    "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
    "start_date": "07-2025"
  }'
```

**Расчёт стоимости за период:**

```bash
curl "http://localhost:8080/api/v1/subscriptions/cost?period_start=01-2025&period_end=12-2025&user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba"
```

**Список подписок с фильтрацией:**

```bash
curl "http://localhost:8080/api/v1/subscriptions?user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba&limit=10&offset=0"
```

## Конфигурация

Настройки задаются через `config.yaml` или переменные окружения:

| Переменная    | Описание                | По умолчанию   |
|---------------|-------------------------|----------------|
| `SERVER_PORT` | Порт HTTP-сервера       | `8080`         |
| `DB_HOST`     | Хост PostgreSQL         | `postgres`     |
| `DB_PORT`     | Порт PostgreSQL         | `5432`         |
| `DB_USER`     | Пользователь БД         | `postgres`     |
| `DB_PASSWORD` | Пароль БД               | `postgres`     |
| `DB_NAME`     | Имя базы данных         | `subscriptions`|
| `DB_SSLMODE`  | Режим SSL               | `disable`      |
| `LOG_LEVEL`   | Уровень логирования     | `info`         |
