# Chat Trace Viewer

Веб-инструмент для просмотра истории одного чата по логам из VictoriaLogs.

Приложение:

- поднимает локальный HTTP-сервер;
- запрашивает сырые записи из VictoriaLogs;
- нормализует события в таймлайн;
- показывает аномалии и диагностическую сводку в браузере.

## Требования

- Go `1.26+`
- доступ к VictoriaLogs

## Конфигурация

Базовые значения лежат в `config/app.env`.
Локальные переопределения можно положить в `config/app.local.env`, взяв за основу `config/app.local.env.example`.

Основные переменные:

- `APP_ADDR` - адрес HTTP-сервера, по умолчанию `127.0.0.1:8080`
- `VICTORIALOGS_BASE_URL` - базовый URL VictoriaLogs
- `VICTORIALOGS_ACCOUNT_ID` - значение заголовка `AccountID`
- `VICTORIALOGS_PROJECT_ID` - значение заголовка `ProjectID`
- `VICTORIALOGS_USERNAME` - логин для basic auth, если нужен
- `VICTORIALOGS_PASSWORD` - пароль для basic auth, если нужен
- `TRACE_DEFAULT_LOOKBACK` - окно поиска по умолчанию, например `30d`
- `TRACE_MAX_LOG_LINES` - лимит строк, читаемых из источника
- `TRACE_MAX_RAW_LINES` - лимит строк, отдаваемых в UI

Пример:

```env
APP_ADDR=127.0.0.1:8080
VICTORIALOGS_BASE_URL=http://localhost:9428
VICTORIALOGS_ACCOUNT_ID=0
VICTORIALOGS_PROJECT_ID=11
VICTORIALOGS_USERNAME=
VICTORIALOGS_PASSWORD=
TRACE_DEFAULT_LOOKBACK=30d
TRACE_MAX_LOG_LINES=500
TRACE_MAX_RAW_LINES=500
```

## Локальный запуск

```bash
cp config/app.local.env.example config/app.local.env
make run
```

После запуска открой `http://127.0.0.1:8080`.

## Проверка

```bash
make test
make build
```

## Структура

- `cmd/chat-trace-viewer` - точка входа приложения
- `internal/httpapi` - HTTP API и раздача веб-статик
- `internal/service` - сценарий сборки chat trace
- `internal/parser` - разбор сырых логов
- `internal/normalizer` - нормализация событий
- `internal/timeline` - сборка таймлайна и аномалий
- `internal/victorialogs` - клиент VictoriaLogs
- `web` - простой фронтенд без сборки
