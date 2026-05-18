# Раздел «Администрирование» — инструкция для фронтенда

## Что такое раздел Admin

Раздел **Admin** — это отдельная часть интерфейса для управления компаниями и пользователями системы.
Он доступен **только пользователям с флагом `isSuperAdmin = true`**.

---

## Авторизация и определение роли

### Ответ на логин

`POST /api/v1/auth/login` возвращает:

```json
{
  "code": 200,
  "message": "Успешно выполнено",
  "payload": {
    "accessToken": "eyJ...",
    "accessTokenExpiresAt": "2025-05-18T11:00:00Z",
    "tokenType": "Bearer",
    "userId": 42,
    "userName": "doctor_ali",
    "companyId": 1
  }
}
```

> `companyId = 0` означает, что суперадмин не привязан к конкретной компании.

`isSuperAdmin` **не возвращается в теле ответа логина** — он находится внутри JWT-токена.

### Как получить `isSuperAdmin` из токена

```js
const payload = JSON.parse(atob(accessToken.split('.')[1]));
const isSuperAdmin = payload.is_super_admin; // boolean, snake_case
```

> Поле в JWT-payload: `is_super_admin` (snake_case).
> Поле в теле запросов/ответов API: `isSuperAdmin` (camelCase).

### Показывать кнопку Admin

```jsx
// React
{isSuperAdmin && <NavItem to="/admin">Администрирование</NavItem>}
```

```vue
<!-- Vue -->
<NavItem v-if="isSuperAdmin" to="/admin">Администрирование</NavItem>
```

### Заголовок для всех запросов

```
Authorization: Bearer <accessToken>
```

Если токен не принадлежит суперадмину → **403 Forbidden**.

### Обновление токена

Когда `accessTokenExpiresAt` прошёл — вызвать refresh:

```
POST /api/v1/auth/refresh
```

Refresh-токен хранится в HTTP-only cookie (устанавливается сервером автоматически).
Запросы к refresh **должны включать cookies** (`credentials: 'include'` / `withCredentials: true`).

---

## ⚠️ Важно: несоответствие регистра поля companyId

В API есть **непоследовательность** — будь внимателен:

| Где                      | Поле в JSON  |
|--------------------------|--------------|
| Login response           | `companyId`  |
| UserResponse (список)    | `companyID`  ← capital D |
| CreateUserRequest (тело) | `companyId`  |

При отображении данных пользователя читай `user.companyID`, при создании — пиши `companyId`.

---

## Доступные роли пользователей

При создании пользователя в поле `role.roleCode` передавай один из:

| Код                  | Название           | Права                                         |
|----------------------|--------------------|-----------------------------------------------|
| `owner`              | Owner              | Полный доступ к компании                      |
| `accountant`         | Accountant         | Бухгалтерия, отчёты, чтение пользователей     |
| `sales_manager`      | Sales Manager      | Продажи, отчёты, чтение пользователей         |
| `warehouse_operator` | Warehouse Operator | Склад, отчёты, чтение пользователей           |

---

## Управление компаниями

Base URL: `/api/v1/admin/company`

### GET — список компаний

```
GET /api/v1/admin/company?name=&status=&limit=20&offset=0
```

| Параметр | Тип    | Описание                              |
|----------|--------|---------------------------------------|
| `name`   | string | Фильтр по названию (ILIKE, частичный) |
| `status` | string | `active` или `inactive`               |
| `limit`  | int    | По умолчанию 20, макс 200             |
| `offset` | int    | По умолчанию 0                        |

**Ответ 200:**
```json
{
  "code": 200,
  "message": "Успешно выполнено",
  "payload": [
    {
      "id": 1,
      "name": "ООО МедЦентр",
      "status": "active",
      "createdAt": "2025-05-18T10:00:00Z",
      "updatedAt": "2025-05-18T10:00:00Z"
    }
  ]
}
```

> Если компаний нет — `payload` будет `null`, не `[]`. Обрабатывай оба варианта.

---

### POST — создать компанию

```
POST /api/v1/admin/company
Content-Type: application/json
```

```json
{
  "name": "ООО МедЦентр",
  "status": "active"
}
```

| Поле     | Тип    | Обязательный | Описание                             |
|----------|--------|:------------:|--------------------------------------|
| `name`   | string | **да**       | Название компании                    |
| `status` | string | нет          | `active` (по умолчанию) / `inactive` |

**Ответ 200:** объект CompanyResponse (см. выше).

---

### PUT — обновить компанию

```
PUT /api/v1/admin/company/:id
Content-Type: application/json
```

Partial update — передавай только изменяемые поля:

```json
{ "name": "Новое название" }
```
```json
{ "status": "inactive" }
```
```json
{ "name": "Новое название", "status": "active" }
```

**Ответы:** `200` CompanyResponse / `400` невалидный статус / `404` не найдена.

---

### DELETE — удалить компанию (soft delete)

```
DELETE /api/v1/admin/company/:id
```

Компания **не удаляется физически** — устанавливается `deleted_at` в БД.
После удаления она не появляется в списках.

**Ответ 200:**
```json
{ "code": 200, "message": "Успешно выполнено", "payload": null }
```

**Ответы:** `200` / `404` не найдена или уже удалена.

---

## Управление пользователями (admin)

Base URL: `/api/v1/admin/users`

> Отличие от `GET/POST /api/v1/users`: суперадмин не ограничен scope своей компании — видит и создаёт пользователей в **любой** компании.

---

### GET — список пользователей

```
GET /api/v1/admin/users?company_id=1&status=active&limit=20&offset=0
```

| Параметр     | Тип    | Описание                                          |
|--------------|--------|---------------------------------------------------|
| `company_id` | int    | Фильтр по компании                                |
| `status`     | string | `active`, `blocked`, `invited`, `deleted`         |
| `username`   | string | Фильтр по логину                                  |
| `first_name` | string | Фильтр по имени                                   |
| `last_name`  | string | Фильтр по фамилии                                 |
| `phone`      | string | Фильтр по телефону                                |
| `email`      | string | Фильтр по email                                   |
| `limit`      | int    | По умолчанию 20                                   |
| `offset`     | int    | По умолчанию 0                                    |

**Ответ 200:**
```json
{
  "code": 200,
  "message": "Успешно выполнено",
  "payload": [
    {
      "id": 5,
      "companyID": 1,
      "userName": "doctor_ali",
      "phone": "+992900000001",
      "email": "ali@clinic.tj",
      "firstName": "Али",
      "lastName": "Назаров",
      "status": "active",
      "lastLoginAt": "2025-05-18T09:00:00Z",
      "createdAt": "2025-05-01T10:00:00Z",
      "updatedAt": "2025-05-18T09:00:00Z"
    }
  ]
}
```

> `isSuperAdmin` **не возвращается** в UserResponse — только в JWT токене пользователя.

---

### POST — создать пользователя

```
POST /api/v1/admin/users
Content-Type: application/json
```

```json
{
  "companyId": 1,
  "userName": "doctor_ali",
  "firstName": "Али",
  "lastName": "Назаров",
  "phone": "+992900000001",
  "email": "ali@clinic.tj",
  "password": "SecurePass123!",
  "status": "active",
  "isSuperAdmin": false,
  "role": {
    "roleCode": "owner",
    "scopeType": "company",
    "scopeId": 1,
    "ownOnly": false
  }
}
```

| Поле           | Тип    | Обязательный | Описание                          |
|----------------|--------|:------------:|-----------------------------------|
| `companyId`    | int    | **да**       | ID компании пользователя          |
| `userName`     | string | **да**       | Уникальный логин                  |
| `firstName`    | string | **да**       | Имя                               |
| `password`     | string | **да**       | Пароль (минимум 8 символов)       |
| `role`         | object | **да**       | Роль и область видимости          |
| `lastName`     | string | нет          |                                   |
| `phone`        | string | нет          |                                   |
| `email`        | string | нет          |                                   |
| `status`       | string | нет          | По умолчанию `active`             |
| `isSuperAdmin` | bool   | нет          | По умолчанию `false`              |

**Объект `role`:**

| Поле        | Тип    | Обязательный | Описание                                                      |
|-------------|--------|:------------:|---------------------------------------------------------------|
| `roleCode`  | string | **да**       | Код роли: `owner`, `accountant`, `sales_manager`, `warehouse_operator` |
| `scopeType` | string | нет          | `company` (по умолчанию), `branch`, `warehouse`, `global`    |
| `scopeId`   | int    | условно      | Обязателен для всех scopeType кроме `global`. Обычно = `companyId` |
| `ownOnly`   | bool   | нет          | `true` — пользователь видит только свои записи               |

**Типичный сценарий создания сотрудника компании:**
```json
"role": {
  "roleCode": "sales_manager",
  "scopeType": "company",
  "scopeId": 1
}
```

**Ответы:** `200` UserResponse / `400` невалидные данные / `409` логин уже занят.

---

### PUT — обновить пользователя

```
PUT /api/v1/admin/users/:id
Content-Type: application/json
```

Partial update — передавай только изменяемые поля:

```json
{
  "firstName": "Алишер",
  "status": "blocked"
}
```

| Поле        | Тип    | Описание                                         |
|-------------|--------|--------------------------------------------------|
| `firstName` | string | Имя                                              |
| `lastName`  | string | Фамилия                                          |
| `phone`     | string | Телефон                                          |
| `email`     | string | Email                                            |
| `status`    | string | `active`, `blocked`, `invited`, `deleted`        |

> Изменение пароля — отдельный эндпоинт (в разработке).
> Изменение роли/scope — отдельный эндпоинт (в разработке).

**Ответы:** `200` UserResponse / `400` невалидный статус / `404` не найден.

---

### DELETE — удалить пользователя (soft delete)

```
DELETE /api/v1/admin/users/:id
```

Устанавливает `status = 'deleted'`. Пользователь **не может войти** в систему.
Все его записи (продажи, закупки и т.д.) сохраняются.

**Ответы:** `200` / `404` не найден или уже удалён.

---

## Статусы пользователя

| Статус    | Описание                                    | Может войти |
|-----------|---------------------------------------------|:-----------:|
| `active`  | Активен                                     | ✅          |
| `blocked`  | Заблокирован администратором               | ❌          |
| `invited` | Приглашён, ещё не принял приглашение        | ❌          |
| `deleted` | Soft-удалён                                 | ❌          |

---

## Общая структура ответа

```json
{
  "code": 200,
  "message": "Успешно выполнено",
  "payload": { ... }
}
```

**Все коды ошибок:**

| HTTP | Значение                                                       |
|------|----------------------------------------------------------------|
| 400  | Некорректные данные (невалидный статус, пустые обязательные поля) |
| 401  | Токен не передан, истёк или невалиден                          |
| 403  | Нет прав (`isSuperAdmin = false`)                              |
| 404  | Объект не найден или уже soft-удалён                           |
| 409  | Конфликт (например, `userName` уже занят)                     |
| 429  | Слишком много запросов (rate limit на `/auth/login`)           |
| 500  | Внутренняя ошибка сервера                                      |

---

## Рекомендуемая структура навигации

```
/admin
  /admin/companies          → список компаний
  /admin/companies/new      → создать компанию
  /admin/companies/:id/edit → редактировать компанию
  /admin/users              → список пользователей (фильтр по компании)
  /admin/users/new          → создать пользователя
  /admin/users/:id/edit     → редактировать пользователя
```

---

## Что НЕ реализовано на бэкенде (в разработке)

| Функционал                        | Статус          |
|-----------------------------------|-----------------|
| Изменение пароля пользователя     | Не реализовано  |
| Изменение роли/scope пользователя | Не реализовано  |
| Список ролей (GET /admin/roles)   | Не реализовано  |
| Восстановление удалённой компании | Не реализовано  |
| Пагинация: total count в ответе   | Не реализовано  |

---

## Swagger UI

Интерактивная документация: `GET /swagger`

Разделы: **Admin / Companies**, **Admin / Users**, **Auth**, **Users**, **Products**, **Sales**, **Purchases**, **Warehouse**.
