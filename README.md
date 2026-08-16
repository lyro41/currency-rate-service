# Currency Rate Service

HTTP-сервис для асинхронного получения и хранения курсов валют. Курс запрашивается через Frankfurter API, сохраняется в PostgreSQL и доступен через HTTP API.

## Возможности

- асинхронное обновление курса через фоновый worker;
- получение результата запроса по UUID;
- получение последнего курса по паре;
- статусы операции: `pending`, `fetched`, `failed`;
- идемпотентные повторные запросы через `Idempotency-Key`;
- автоматическое создание таблицы и индексов PostgreSQL при запуске.

## Требования

- Go 1.26+;
- PostgreSQL 12+;
- доступ к `api.frankfurter.dev` по HTTPS.

## Конфигурация

YAML-файл конфигурации является необязательным. Если `CONFIG_PATH` задан, сервис читает настройки из указанного файла. Пример `config/local.yaml` (конфиг по умолчанию):

```yaml
env: development
storage:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: postgres
  timeout: 5s
provider:
  timeout: 5s
http_server:
  address: ":8080"
  read_timeout: 5s
  write_timeout: 10s
  idle_timeout: 60s
worker:
  buffer_size: 100
```

Если `CONFIG_PATH` задан, указанный файл должен существовать и содержать корректный YAML-конфиг.

## Сборка и запуск

```bash
go mod download
go test ./...
go build -o build/currency-rate-service ./cmd/server
export CONFIG_PATH="$PWD/config/local.yaml"
./build/currency-rate-service
```

В Windows PowerShell:

```powershell
New-Item -ItemType Directory -Force config, build
$env:CONFIG_PATH = "$PWD/config/local.yaml"
go run .\cmd\server
```

При старте сервис подключается к PostgreSQL и выполняет `internal/db/sql/schema.sql` автоматически. Отдельный запуск миграций не требуется.

### Запуск через Docker Compose

Для запуска приложения и PostgreSQL выполните:

```bash
docker compose up --build
```

После запуска API доступен по адресу `http://localhost:8080`. Остановить контейнеры можно командой:

```bash
docker compose down
```

Данные PostgreSQL сохраняются в Docker volume `postgres-data`.

## OpenAPI / Swagger

Спецификация API находится в файле [openapi.yaml](openapi.yaml). Её можно открыть в [Swagger Editor](https://editor.swagger.io/) или импортировать в Swagger UI/Postman.

Если `CONFIG_PATH` не задан, конфигурация читается через `cleanenv` из переменных окружения и получает значения по умолчанию из тегов `env-default`. Поэтому сервис можно запустить без YAML-файла:

```bash
go run ./cmd/server
```

Чтобы использовать собственные настройки, задайте `CONFIG_PATH` и укажите путь к YAML-файлу.

## API

Поддерживаемые валюты: `USD`, `EUR`, `MXN`, `RUB`, `JPY`, `AMD`. Пара передаётся в формате `ABC/XYZ`, например `USD/RUB`.

### Запустить обновление

```http
POST /update/USD/RUB
```

```bash
curl -X POST http://localhost:8080/update/USD/RUB
```

Ответ содержит UUID операции:

```json
{"pair":"USD/RUB","id":"8d8f5e8d-5f1b-4f5e-bf35-3de5c89a1f20"}
```

Операция сначала получает статус `pending`, а затем обрабатывается worker’ом.
Очередь имеет ограниченный размер; при её переполнении сервис возвращает `503 Service Unavailable`.

### Идемпотентное обновление

Передайте заголовок `Idempotency-Key` (длиной не более 255 символов):

```bash
curl -X POST http://localhost:8080/update/USD/RUB `
  -H "Idempotency-Key: update-usd-rub-2026-08-15"
```

Повторный запрос с тем же ключом и той же парой вернёт тот же UUID и не создаст новую задачу. Один ключ может использоваться независимо для разных валютных пар. Без заголовка каждый запрос создаёт новую операцию.

### Получить результат по UUID

```http
GET /currency-rate?id=<uuid>
```

```bash
curl "http://localhost:8080/currency-rate?id=8d8f5e8d-5f1b-4f5e-bf35-3de5c89a1f20"
```

Пример ответа:

```json
{
  "pair": "USD/RUB",
  "id": "8d8f5e8d-5f1b-4f5e-bf35-3de5c89a1f20",
  "rate": "92.1234",
  "time": "2026-08-15T12:00:00Z",
  "status": "fetched"
}
```

### Получить последний успешный курс

```http
GET /currency-rate/USD/RUB
```

```bash
curl http://localhost:8080/currency-rate/USD/RUB
```

### Ошибки

Ошибки возвращаются в JSON с полями `error` и `pair`. 

Коды ошибок:
- `400 Bad Request`: некорректный формат пары или UUID возвращает,
- `404 Not Found`: отсутствующая котировка или операция,
- `422 Unprocessable Entity`: неподдерживаемая валюта, 
- `500 Internal Server Error`: ошибки PostgreSQL,
- `503 Service Unavailable`: переполненная очередь. 

## Структура проекта

```text
cmd/server/          точка входа
internal/config/     конфигурация
internal/handlers/   HTTP handlers и валидация
internal/provider/   клиент Frankfurter API
internal/worker/     фоновая обработка
internal/db/         sqlc-код и SQL-схема
internal/api/        модели API и статусы
```

## Разработка

После изменения SQL-запросов сгенерируйте код заново командой `sqlc generate`. Перед отправкой изменений выполните:

```bash
gofmt -w .
go test ./...
go vet ./...
```

Лицензия в проекте не задана.
