# NurMed

## Запуск в Docker
- Собрать образ: `docker build -t nurmed .`
- Запустить контейнер: `docker run --rm -p 9090:9090 nurmed`
- Или через docker compose (удобно для локалки/CI): `docker compose up --build`
- Конфиг берется из `configs/appConfig.json`, он монтируется в контейнер через compose и может переопределяться переменными окружения (`SERVER_PORT`, `SERVER_HOST`, `SERVER_VERSION` и т.д.).
- После старта хелсчек доступен по `GET http://localhost:9090/api/v1/health`.
- Логи пишутся в `/app/app.log` внутри контейнера; для сохранения на хост можно примонтировать файл/папку или читать stdout.

## Swagger
- Swagger UI: `GET http://localhost:9050/swagger`
- OpenAPI spec: `GET http://localhost:9050/swagger/openapi.yaml`

## Документация для фронта
- Авторизация/аутентификация: `docs/AUTH_FRONTEND_RU.md`
- Роли/permissions/scope: `docs/RBAC_SCOPE_RU.md`
