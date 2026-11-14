# aura-spaceapps-frontend

Веб-интерфейс AURA. Общий контекст продукта — `../CLAUDE.md`, требования — `../docs/REQUIREMENTS.md`, план — `../docs/PLAN.md`.

## Стек

- React 19 + TypeScript 7 + Vite 8
- Tailwind CSS v4 (токены в `src/index.css`, блок `@theme`)
- React Router v7 (data router, `src/app/router.tsx`)
- TanStack Query — всё серверное состояние
- Recharts — графики трендов
- Vitest + Testing Library — тесты; oxlint — линтер

## Команды

```bash
npm run dev          # :5173, /api/* проксируется на Go-бэкенд :8080
npm run build        # tsc -b + vite build
npm run typecheck
npm run lint         # oxlint
npm test             # vitest run
npm run gen:api      # типы из ../aura-spaceapps-backend/api/openapi.yaml → src/shared/api/schema.d.ts
```

Перед тем как считать задачу готовой: `npm run build`, `npm test`, `npm run lint` — всё чисто.

`gen:api` запускает openapi-typescript через `npx`, т.к. он пока требует TypeScript 5 и не ставится рядом с TS 7 в devDependencies.

## Структура

```
src/
  app/          корень: App (провайдеры), router, Layout (навигация)
  pages/        экраны — по одному на маршрут; только компоновка фич
  features/     фичи: readiness, body-map, measurements, trends, guidance, tasks
                  <feature>/api.ts         хуки useQuery/useMutation
                  <feature>/components/    UI фичи
                  <feature>/*.test.ts(x)
  shared/
    api/        client.ts (fetch-обёртка), schema.d.ts (СГЕНЕРИРОВАН — не править руками)
    lib/        чистые утилиты: status.ts (цветовой язык статусов), форматирование
    ui/         переиспользуемые компоненты без бизнес-логики
  test/         setup для vitest
```

Папка `features/` появляется с первой фичей. Фича не импортирует другую фичу напрямую — общее выносится в `shared/`.

## Правила

- **Типы API** — только из `schema.d.ts` через `Schemas['Name']` (`shared/api/client.ts`). Руками интерфейсы ответов не пишем.
- **Серверное состояние** — только TanStack Query. Ключи запросов — массивы, начинающиеся с имени ресурса: `['measurements', astronautId]`. Никакого Redux/Zustand, пока не появится реальная нужда в клиентском глобальном состоянии.
- **Статусы и цвета** — только через `shared/lib/status.ts` и токены `status-*`. Не хардкодь `green-500` и т.п. для статусов.
- **Доступность:** статус никогда не передаётся только цветом — всегда плюс текст или иконка (дальтонизм).
- **Offline:** никаких CDN, Google Fonts, внешних картинок — всё кладём в бандл.
- **Body map** — inline SVG-компонент, каждая система организма — отдельный `<path>`/`<g>` с `id` = код системы из API.
- UI на английском. Тёмная «космическая» тема по умолчанию.
- Компоненты — функциональные, именованный экспорт (`export function X`), один компонент на файл в `components/`.
