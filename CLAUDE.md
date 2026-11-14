# aura-spaceapps-frontend

Веб-интерфейс AURA (Astronaut Universal Readiness Assessment) — инструмент самооценки здоровья астронавта, NASA Space Apps 2026. API — репозиторий `aura-spaceapps-backend`, клонируется рядом.

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
- **Статусы и цвета** — только через `shared/lib/status.ts` (`alertClassOf`, `alertBg`, `alertText`) и токены. Не хардкодь `red-500`, `green-500` и т.п.
- **Доступность:** класс тревоги никогда не передаётся только цветом — только через `shared/ui/StatusBadge` (глиф формы + слово).
- **Offline:** никаких CDN, Google Fonts, внешних картинок — всё в бандле. Шрифты — `@fontsource/inter` и `@fontsource/dm-mono`, подключены в `main.tsx`.
- **Body map** — inline SVG-компонент, каждая система организма — отдельный `<path>`/`<g>` с `id` = код системы из API. Без декоративной графики.
- UI на английском. Компоненты — функциональные, именованный экспорт, один компонент на файл.

## Дизайн: NASA-STD-3001 Vol. 2, Appendix F

Интерфейс следует стандарту NASA на дисплеи экипажа: https://www.nasa.gov/reference/appendix-f-vol-2/ (в скобках — разделы). Токены — `src/index.css` (`@theme`), зеркало брендбука AURA v2; значения меняются сначала в брендбуке.

- **Цвет только для смысла, без декора** (F.5.10). Никаких декоративных полос, узоров, градиентов, теней.
- **Таблица цветов NASA (F.5-1):** `warning` красный (Emergency/Warning), `caution` жёлтый, `advisory` синий, `stale` циан (нет/устаревшие данные), `unavailable` серый (disabled). Текст на заливке — `on-alert`; классом как текстом — `*-ink`.
- **Рекомендованные (F.5-2):** `command` тёмно-зелёный фон кнопок-команд (+ `on-command`, рамка `command-edge`); `navigation` ярко-зелёный — выбранная вкладка и фокус. **Синий — только Advisory**, зелёный — только команды/навигация.
- **Норма не раскрашивается:** nominal = `text-ink`. Цветом выделяется только отклонение; к значению вне нормы — стрелка ▲/▼ цветом класса сразу после числа (F.4.3.1).
- **Классы:** backend `critical` → Warning, `caution` → Caution, `unknown` → Stale, `nominal` → Nominal; советы → Advisory.
- **Реверс** (заливка всей плитки) — только Warning. Мигание не используем (F.5.11).
- **Контраст текста ≥ 6:1** во всех темах — новые цвета проверять по WCAG-формуле.
- **Шрифты:** Inter — текст; **DM Mono — все числа, время, коды, ID** (F.5.1). Минимум 14px. **Смешанный регистр**, капсом только аббревиатуры (EVA, MET, SYS-01). Подпись поля слева или над значением.
- **Компоновка (F.4.2):** навигация сверху; постоянная статус-зона в шапке (`app/Layout.tsx`): MET, UTC, связь AOS/LOS, задержка до Земли (+ счётчики тревог, когда появится выбранный астронавт). Уникальный заголовок экрана слева сверху (40px).
- **Время (F.6.1):** всегда с подписью системы: `MET 209/14:08:33`, `UTC 09:08:33` — `shared/lib/time.ts`.
- **Графики (F.5.8):** заголовок, оси с единицами, легенда, пороги линиями с подписью, вне нормы — цвет + число.
- **Рекомендации (F.6.2):** всегда с Basis; тексты без «I think»; Acknowledge.
- Необратимые действия — только с подтверждением (F.4.1).
