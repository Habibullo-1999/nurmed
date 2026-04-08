# Swagger Setup

## Доступ к Swagger UI
После запуска приложения, Swagger UI доступен по адресу:
- `http://localhost:YOUR_PORT/swagger`

## Как это работает
1. Swagger UI загружается из CDN (swagger-ui-dist@5)
2. OpenAPI spec загружается из `/swagger/openapi.yaml`
3. Все эндпоинты документированы в `api/openapi/openapi.yaml`

## Трубулшутинг
- Если Swagger не загружается - проверьте, что приложение запущено
- Если "не видно" эндпоинтов - обновите `api/openapi/openapi.yaml` с правильными путями и определениями
- CORS должен быть правильно настроен для доступа к openapi.yaml

