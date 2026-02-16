# RBAC + Scope (роли, permissions, company/branch/warehouse)

## 1) Что это и зачем
В проекте доступ к данным и действиям строится по модели:
- `RBAC` (Role-Based Access Control): роль дает набор прав.
- `Scope`: ограничивает область действия прав.

Простыми словами:
- Роль отвечает за `что можно делать`.
- Scope отвечает за `где именно это можно делать`.

## 2) Основные сущности

- `roles`
  - Например: `owner`, `accountant`, `sales_manager`, `warehouse_operator`.
- `permissions`
  - Формат: `module.resource.action`.
  - Примеры: `users.read`, `users.create`, `warehouse.stock.update`.
- `role_permissions`
  - Связка: какие permissions входят в роль.
- `user_roles`
  - Назначение роли пользователю + scope:
    - `scope_type`: `global | company | branch | warehouse`
    - `scope_id`: ID компании/филиала/склада (для `global` пусто)
    - `own_only`: только свои записи.

## 3) Как читать scope

- `global`
  - Доступ без ограничения по company/branch/warehouse.
- `company`
  - Доступ только в пределах компании `scope_id`.
- `branch`
  - Доступ только в пределах филиала `scope_id`.
- `warehouse`
  - Доступ только в пределах склада `scope_id`.
- `own_only=true`
  - Даже при наличии permission пользователь может работать только со своими данными.

## 4) Что проверяет backend на каждом защищенном запросе

1. `AuthMiddleware`
- Проверяет `Authorization: Bearer <accessToken>`.
- Проверяет, что пользователь активен (`status=active`).

2. `ScopeMiddleware`
- Читает scope из запроса:
  - query: `company_id`, `branch_id`, `warehouse_id`, `owner_user_id`
  - headers: `X-Company-ID`, `X-Branch-ID`, `X-Warehouse-ID`, `X-Owner-User-ID`
- Сверяет с `user_roles` пользователя.

3. `PermissionMiddleware("...")`
- Проверяет, есть ли нужное право у пользователя с учетом scope.

Если хоть одна проверка не прошла:
- `401` (нет/невалидный токен)
- `403` (нет прав или scope не подходит)

## 5) Примеры из текущего API

- `GET /api/v1/users`
  - Требует `users.read`
  - + scope ограничение

- `POST /api/v1/users`
  - Требует `users.create`
  - + scope ограничение

## 6) Как назначать доступ пользователю

### Вариант A: одна роль на компанию
- role: `accountant`
- scope: `company`
- scope_id: `1`

Результат:
- Пользователь работает как бухгалтер только в компании `1`.

### Вариант B: одна роль на филиал
- role: `sales_manager`
- scope: `branch`
- scope_id: `22`

Результат:
- Доступ только к филиалу `22`.

### Вариант C: owner на всю систему
- role: `owner`
- scope: `global`

Результат:
- Полный доступ (если не ограничен другими бизнес-правилами).

## 7) Что передавать с фронта

Для защищенных API всегда:
- `Authorization: Bearer <accessToken>`

Для контекста данных (по необходимости):
- `X-Company-ID: <id>`
- `X-Branch-ID: <id>`
- `X-Warehouse-ID: <id>`
- `X-Owner-User-ID: <id>`

Важно:
- Скрывать кнопки на фронте полезно для UX.
- Но реальная защита всегда только на backend.

## 8) Рекомендации по моделированию прав

- Делайте permission мелкими и явными:
  - хорошо: `warehouse.stock.update`
  - плохо: `warehouse.manage_all`
- Для критичных действий используйте отдельные permissions:
  - `refund.approve`, `stock.writeoff`, `period.close`
- Любые критичные изменения пишите в `audit_logs`.

## 9) Минимальный процесс запуска в новой компании

1. Создать запись компании (`company`).
2. Создать филиалы (`branch`) и склады (`warehouse`) если нужно.
3. Создать роли/permissions (или использовать стандартные).
4. Создать пользователей.
5. Назначить `user_roles` с нужным `scope_type/scope_id`.
6. Проверить доступ тестами:
- успешный сценарий (200)
- запрет по permission (403)
- запрет по scope (403)

## 10) Частые ошибки

- Назначили роль, но забыли `role_permissions`.
- Назначили `scope_type=company`, но `scope_id` не передали.
- На фронте не отправляется `Authorization`.
- Пытаются защититься только скрытием кнопок на UI.
