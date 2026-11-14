# aura-backend

Go API для AURA. Общий контекст продукта — `../CLAUDE.md`, требования — `../docs/REQUIREMENTS.md`, план — `../docs/PLAN.md`.

## Стек

- Go 1.26+, только стандартная библиотека для HTTP (`net/http` с паттернами `"GET /path/{id}"`), без фреймворков.
- SQLite через `modernc.org/sqlite` (pure Go, без CGO) — БД это один файл, работает офлайн.
- Логи — `log/slog`. Конфиг — переменные окружения (`internal/config`, см. `.env.example`).

## Команды

```bash
go mod tidy                 # после добавления/удаления импортов
go run ./cmd/aura           # сервер на :8080
go test ./...               # все тесты
go vet ./...
gofmt -w .
```

Перед тем как считать задачу готовой: `gofmt`, `go vet ./...`, `go test ./...` — всё чисто.

## Архитектура

```
cmd/aura/            main: конфиг → БД → HTTP-сервер, graceful shutdown
api/openapi.yaml     контракт API (источник правды для фронта)
internal/
  config/            env → Config
  domain/            типы и каталог показателей; без I/O, ни от чего не зависит
  readiness/         показатели → статусы → Readiness Score; чистые функции
  guidance/          оценка → рекомендации и эскалация; чистые функции
  storage/           SQLite: подключение, миграции, репозитории
    migrations/      NNNN_name.sql, встраиваются в бинарник через embed
  seed/              генерация демо-данных (детерминированная)
  httpapi/           роутинг, middleware, JSON, валидация; тонкие хендлеры
```

Направление зависимостей: `httpapi → readiness, guidance, storage → domain`. `readiness` и `guidance` не знают про БД и HTTP — это позволяет покрыть всю медицинскую логику табличными юнит-тестами.

## Соглашения

- **API:** все маршруты под `/api/v1`. JSON-поля — `camelCase`. Коды показателей и enum-значения — `snake_case` (`heart_rate`, `bp_systolic`). Ошибки всегда в формате `{"error": "..."}` через `writeError`.
- **Время:** в БД — TEXT в ISO-8601 UTC; в Go — `time.Time` в UTC; в JSON — RFC 3339.
- **Ошибки:** оборачивай с префиксом пакета: `fmt.Errorf("storage: insert measurement: %w", err)`. Не логируй и возвращай одновременно — логирует тот, кто обрабатывает.
- **Миграции:** только добавлять новые файлы со следующим номером. Уже существующую миграцию не редактировать.
- **Тесты:** табличные (`[]struct{name string; ...}`), для БД — временный файл через `t.TempDir()`, для HTTP — `httptest`. Пороги и правила в `readiness`/`guidance` — обязательно с тестами на границы диапазонов.
- **Пороговые значения:** константы в `readiness` с комментарием-источником (`// Source: NASA HRP ...`).
- Новые зависимости — только если стандартной библиотеки реально не хватает; обсуди перед добавлением.

## Как добавить эндпоинт

1. Опиши путь и схемы в `api/openapi.yaml`.
2. Если нужны новые таблицы — новая миграция в `internal/storage/migrations/`.
3. Метод репозитория в `internal/storage`, бизнес-логика в `readiness`/`guidance`.
4. Хендлер в `internal/httpapi/<resource>.go`, регистрация в `router.go`, тест в `<resource>_test.go`.
5. Сообщи фронту: им нужно выполнить `npm run gen:api`.
