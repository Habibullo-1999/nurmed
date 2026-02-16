# Авторизация и аутентификация для фронта (простыми словами)

## 1) Как это работает в целом
- Аутентификация отвечает на вопрос: `кто пользователь`.
- Авторизация отвечает на вопрос: `что пользователю можно делать`.

В проекте это сделано так:
- Логин: `username/email/phone + password`.
- После логина сервер дает:
  - `accessToken` (короткий, ~15 минут) в JSON ответе.
  - `refresh_token` в `HttpOnly cookie` (длинный, ~30 дней).
- Когда `accessToken` истек, фронт вызывает `/auth/refresh` и получает новый `accessToken`.
- Права проверяются на backend по `permission` + `scope` (company/branch/warehouse).

## 2) Базовые URL
- API base: `http://localhost:9050/api/v1`
- Swagger UI: `http://localhost:9050/swagger`
- OpenAPI: `http://localhost:9050/swagger/openapi.yaml`

## 3) Что нужно настроить на фронте
- Хранить `accessToken` (лучше в памяти приложения).
- Для запросов refresh/logout отправлять cookie:
  - `fetch`: `credentials: 'include'`
  - `axios`: `withCredentials: true`
- В каждый защищенный запрос добавлять заголовок:
  - `Authorization: Bearer <accessToken>`
- Для scope (если нужно) добавлять заголовки:
  - `X-Company-ID`, `X-Branch-ID`, `X-Warehouse-ID`, `X-Owner-User-ID`

## 4) Эндпоинты аутентификации

### POST `/auth/login`
Тело:
```json
{
  "identifier": "owner",
  "password": "StrongPass123!"
}
```
Ответ (`payload`):
```json
{
  "accessToken": "...",
  "accessTokenExpiresAt": "2026-02-16T12:00:00Z",
  "tokenType": "Bearer",
  "userId": 1,
  "userName": "owner"
}
```
Важно:
- Сервер также установит cookie `refresh_token` (HttpOnly).
- На логин стоит rate limit (по IP): по умолчанию `20` запросов в минуту.

### POST `/auth/refresh`
- Обычно тело не нужно.
- Достаточно отправить запрос с cookie (`credentials: include`).
- Альтернатива для не-браузерных клиентов: `X-Refresh-Token` header.

Ответ: новый `accessToken` + новая refresh cookie (ротация).

### POST `/auth/logout`
- Отзывает refresh-сессию и очищает refresh cookie.

## 5) Защищенные API

### GET `/users`
Требует:
- `Authorization: Bearer <accessToken>`
- Право: `users.read`

Опционально:
- `X-Company-ID` (или query `company_id`) для scope.

### POST `/users`
Требует:
- `Authorization: Bearer <accessToken>`
- Право: `users.create`

Тело:
```json
{
  "companyId": 1,
  "userName": "new_user",
  "phone": "+992900000001",
  "email": "new_user@example.com",
  "password": "StrongPass123!",
  "firstName": "Ali",
  "lastName": "Karimov",
  "status": "active",
  "isSuperAdmin": false,
  "role": {
    "roleCode": "accountant",
    "scopeType": "company",
    "scopeId": 1,
    "ownOnly": false
  }
}
```

## 6) Роли, permissions, scope
- Роль (`roleCode`) = набор прав (`permissions`).
- Permission пример: `users.read`, `users.create`, `warehouse.stock.update`.
- Scope ограничивает, где действует право:
  - `global`
  - `company`
  - `branch`
  - `warehouse`

Если пользователь не super-admin, backend все равно проверит scope даже если фронт не спрятал кнопку.

## 7) Какие ошибки обрабатывать на фронте
Сервер отвечает в формате:
```json
{
  "code": 401,
  "message": "Unauthorized",
  "payload": null
}
```

Типовые коды:
- `400` — неправильные данные
- `401` — токен невалидный/истек
- `403` — нет прав или scope
- `409` — конфликт (например, username/email/phone уже заняты)
- `429` — слишком много попыток логина

## 8) Рекомендуемый flow на фронте
1. Пользователь вводит логин/пароль -> `/auth/login`.
2. Сохранить `accessToken`.
3. Все API дергать с `Authorization`.
4. Если получили `401`:
   - один раз вызвать `/auth/refresh` с `credentials: include`.
   - если refresh успешен, повторить исходный запрос.
   - если refresh неуспешен, отправить пользователя на login.
5. На выходе вызвать `/auth/logout` и очистить локальное auth-состояние.

## 9) Важно для dev-среды
Сейчас в конфиге `auth.secureCookies=true`.
Это значит refresh cookie работает только по HTTPS.

Для локальной разработки на `http://localhost`:
- либо поднимать HTTPS,
- либо временно ставить `auth.secureCookies=false` в `configs/appConfig.json`.

Также для cookie между разными доменами нужно корректно настроить CORS с `AllowCredentials=true` и конкретными `AllowedOrigins`.
