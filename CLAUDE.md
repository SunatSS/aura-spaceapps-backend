# aura-spaceapps-backend

Go API для AURA (Astronaut Universal Readiness Assessment) — инструмент самооценки здоровья астронавта, NASA Space Apps 2026. Веб-интерфейс — репозиторий `aura-spaceapps-frontend`, клонируется рядом.

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
cmd/aura/            main: конфиг → каталог → БД → сид пустой БД → HTTP-сервер, graceful shutdown
api/openapi.yaml     контракт API (источник правды для фронта)
internal/
  config/            env → Config (PORT для Render, AURA_STATIC_DIR, AURA_SEED)
  domain/            Mission/Phase, Astronaut, Measurement, Activity, Status; без I/O
  auth/              хэш паролей (PBKDF2-SHA256, stdlib), токены сессий, политика доступа по ролям
  catalog/           catalog.yaml (встроен в бинарник): показатели, пороги, тексты, риски HRP,
                     среды, системы организма, тексты правил; валидация при загрузке
  readiness/         показатели → статусы → Readiness Score; чистые функции
  guidance/          оценка + история → рекомендации и эскалация; чистые функции
  report/            отчёт для флайт-хирурга (JSON и текст): сводка, тревоги, тренды за 7 дней, воздух модуля
  vehicle/           среда в модулях: телеметрия → статусы по порогам catalog.vehicle_parameters (Stale по свежести)
  supplies/          медсклад: статусы Warning/Caution/Advisory, прогноз расхода до конца миссии
  storage/           SQLite: подключение, миграции, репозитории (Queries, Tx)
    migrations/      NNNN_name.sql, встраиваются в бинарник через embed
  seed/              демо-миссия к Марсу: 30 суток, перелёт → посадка → поверхность с EVA
  httpapi/           роутинг, middleware, JSON, валидация, раздача собранного фронта (SPA)
Dockerfile           собирает фронт из репо aura-spaceapps-frontend + Go-бинарник в один образ
render.yaml          Render Blueprint (бесплатный web service)
```

Направление зависимостей: `httpapi → readiness, guidance, storage → catalog → domain`. `readiness` и `guidance` не знают про БД и HTTP — вся медицинская логика покрыта табличными юнит-тестами.

**Тексты и пороги** правятся только в `internal/catalog/catalog.yaml` (неизвестные поля — ошибка). После правки — `go test ./...`.

**Демо-данные:** при старте с пустой БД (`AURA_SEED=auto`) создаётся миссия MX-1, день 210. Сценарии: Morgan — норма; Sato — растёт давление; Rahimova — недосып; Ferreira — гипоксия после EVA (эскалация). Тест `seed_test.go` проверяет, что сценарии не ломаются.

## Соглашения

- **API:** все маршруты под `/api/v1`. JSON-поля — `camelCase`. Коды показателей и enum-значения — `snake_case` (`heart_rate`, `bp_systolic`). Ошибки всегда в формате `{"error": "..."}` через `writeError`.
- **Время:** в БД — TEXT в ISO-8601 UTC; в Go — `time.Time` в UTC; в JSON — RFC 3339.
- **Ошибки:** оборачивай с префиксом пакета: `fmt.Errorf("storage: insert measurement: %w", err)`. Не логируй и возвращай одновременно — логирует тот, кто обрабатывает.
- **Миграции:** только добавлять новые файлы со следующим номером. Уже существующую миграцию не редактировать.
- **Тесты:** табличные (`[]struct{name string; ...}`), для БД — временный файл через `t.TempDir()`, для HTTP — `httptest`. Пороги и правила в `readiness`/`guidance` — обязательно с тестами на границы диапазонов.
- **Пороговые значения:** в `catalog.yaml` с комментарием-источником (`# Source: ...`); они иллюстративные — ресурсы NASA чисел не дают.
- **Вход и роли:** логин/пароль, сессия — HTTP-only cookie `aura_session` (в БД только SHA-256 токена). Все маршруты, кроме health, indicators, missions/current и auth/login|logout|demo-accounts, обёрнуты в `h.authed`. Доступ к астронавту проверяет `loadAstronaut` (`auth.CanView`), запись замеров — `auth.CanLog`. Демо-аккаунты: фамилия в нижнем регистре, пароль `seed.DemoPassword`. В тестах `e.login("rahimova")`.
- Новые зависимости — только если стандартной библиотеки реально не хватает; обсуди перед добавлением.

## Как добавить эндпоинт

1. Опиши путь и схемы в `api/openapi.yaml`.
2. Если нужны новые таблицы — новая миграция в `internal/storage/migrations/`.
3. Метод репозитория в `internal/storage`, бизнес-логика в `readiness`/`guidance`.
4. Хендлер в `internal/httpapi/<resource>.go`, регистрация в `router.go`, тест в `<resource>_test.go`.
5. Сообщи фронту: им нужно выполнить `npm run gen:api`.
