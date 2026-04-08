# NurMed

## Swagger UI
После запуска приложения откройте в браузере:
```
http://localhost:YOUR_PORT/swagger
```

Убедитесь, что приложение запущено и файл `api/openapi/openapi.yaml` находится в корне проекта.

## Запуск в Docker
- Собрать образ: `docker build -t nurmed .`
- Запустить контейнер: `docker run --rm -p 9090:9090 nurmed`
- Или через docker compose (удобно для локалки/CI): `docker compose up --build`
- Конфиг берется из `configs/appConfig.json`, он монтируется в контейнер через compose и может переопределяться переменными окружения (`SERVER_PORT`, `SERVER_HOST`, `SERVER_VERSION` и т.д.).
- После старта хелсчек доступен по `GET http://localhost:9090/api/v1/health`.
- Логи пишутся в `/app/app.log` внутри контейнера; для сохранения на хост можно примонтировать файл/папку или читать stdout.

## Запуск

```bash
cd /Users/habibullogulomidinov/projects/nurmed
go build ./...
```

```bash
docker build -t nurmed .
docker compose up --build
```

Healthcheck: `GET http://localhost:9050/api/v1/health`.
Swagger: `GET http://localhost:9050/swagger`.
