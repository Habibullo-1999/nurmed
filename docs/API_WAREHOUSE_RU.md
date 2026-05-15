# API — Склад (Warehouse)

Базовый URL: `/api/v1/warehouse`

Все эндпоинты требуют JWT-токена в заголовке:
```
Authorization: Bearer <token>
```

Компания определяется автоматически из токена. Передавать `X-Company-ID` или `company_id` не обязательно — они нужны только суперадмину.

---

## Содержание

1. [Склады (Warehouses)](#1-склады)
2. [Остатки товаров (Stock)](#2-остатки-товаров)
3. [Инвентаризация (Inventory)](#3-инвентаризация)
4. [Перемещение товаров (Transfer)](#4-перемещение-товаров)
5. [Списание товаров (Writeoff)](#5-списание-товаров)
6. [Коды статусов HTTP](#6-коды-статусов-http)
7. [Статусы документов](#7-статусы-документов)

---

## 1. Склады

### GET `/warehouse/warehouses`

Возвращает список активных складов компании. Используется для заполнения выпадающих списков при создании документов.

**Права:** `warehouse.warehouses.read`

**Query-параметры:** нет

**Пример ответа:**
```json
{
  "code": 200,
  "message": "Успешно выполнено",
  "payload": [
    {
      "id": 1,
      "companyID": 1,
      "name": "Основной склад",
      "address": "",
      "status": "active",
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-01T00:00:00Z"
    }
  ]
}
```

**Поля ответа:**

| Поле | Тип | Описание |
|---|---|---|
| `id` | int64 | ID склада |
| `companyID` | int64 | ID компании |
| `name` | string | Название склада |
| `address` | string | Адрес (может быть пустым) |
| `status` | string | `active` / `inactive` |
| `createdAt` | datetime | Дата создания |
| `updatedAt` | datetime | Дата последнего изменения |

---

## 2. Остатки товаров

### GET `/warehouse/stock`

Возвращает текущие остатки товаров по складам. Показывает только позиции с количеством > 0.

**Права:** `warehouse.stock.read`

**Query-параметры:**

| Параметр | Тип | Обязательный | Описание |
|---|---|---|---|
| `warehouse_id` | int64 | нет | Фильтр по конкретному складу |
| `search` | string | нет | Поиск по названию товара (частичное совпадение), точному штрих-коду или артикулу |
| `limit` | int | нет | Количество записей на странице (по умолч. 20, макс. 200) |
| `offset` | int | нет | Смещение для пагинации (по умолч. 0) |

**Пример запроса:**
```
GET /api/v1/warehouse/stock?search=Пикамилон&limit=10&offset=0
```

**Пример ответа:**
```json
{
  "code": 200,
  "message": "Успешно выполнено",
  "payload": [
    {
      "id": 1,
      "warehouseId": 1,
      "warehouseName": "Основной склад",
      "productId": 42,
      "productName": "Пикамилон 20мг №30тб",
      "sku": "",
      "barcode": "4601808015966",
      "unit": "шт",
      "quantity": 1.0,
      "costPrice": 21.00,
      "totalCostPrice": 21.00
    }
  ]
}
```

**Поля ответа:**

| Поле | Тип | Описание |
|---|---|---|
| `id` | int64 | ID записи об остатке |
| `warehouseId` | int64 | ID склада |
| `warehouseName` | string | Название склада |
| `productId` | int64 | ID товара |
| `productName` | string | Название товара (Номенклатура) |
| `sku` | string | Артикул |
| `barcode` | string | Штрих-код |
| `unit` | string | Единица измерения (шт, мл, г и т.д.) |
| `quantity` | float64 | Текущее количество на складе |
| `costPrice` | float64 | Себестоимость единицы |
| `totalCostPrice` | float64 | Общая себестоимость = `quantity × costPrice` |

---

## 3. Инвентаризация

Инвентаризация позволяет сверить фактические остатки на складе с данными системы. После проведения документа (POST) система обновляет остатки на фактические значения.

### GET `/warehouse/inventories`

Список документов инвентаризации.

**Права:** `warehouse.inventory.read`

**Query-параметры:**

| Параметр | Тип | Обязательный | Описание |
|---|---|---|---|
| `warehouse_id` | int64 | нет | Фильтр по складу |
| `status` | string | нет | Фильтр по статусу: `draft`, `posted`, `cancelled` |
| `date_from` | date | нет | Фильтр с даты (формат: `2024-01-01`) |
| `date_to` | date | нет | Фильтр по дату включительно |
| `search` | string | нет | Поиск по номеру документа |
| `limit` | int | нет | Кол-во записей (по умолч. 20) |
| `offset` | int | нет | Смещение |

**Пример ответа:**
```json
{
  "code": 200,
  "message": "Успешно выполнено",
  "payload": [
    {
      "id": 1,
      "companyID": 1,
      "warehouseId": 1,
      "warehouseName": "Основной склад",
      "documentNo": "INV-1718000000-a1b2",
      "status": "posted",
      "surplusDeficit": -12.50,
      "note": "",
      "createdBy": 5,
      "createdByName": "Иван Петров",
      "updatedBy": 5,
      "updatedByName": "Иван Петров",
      "createdAt": "2024-05-01T10:00:00Z",
      "updatedAt": "2024-05-01T11:00:00Z"
    }
  ]
}
```

---

### POST `/warehouse/inventories`

Создаёт новый документ инвентаризации со статусом `draft`. Документ не изменяет остатки — это происходит только при проведении.

**Права:** `warehouse.inventory.create`

**Тело запроса:**
```json
{
  "warehouseId": 1,
  "documentNo": "",
  "note": "Плановая инвентаризация за май",
  "items": [
    {
      "productId": 42,
      "productName": "Пикамилон 20мг №30тб",
      "expectedQty": 10.0,
      "actualQty": 8.0,
      "costPrice": 21.00
    }
  ]
}
```

**Поля запроса:**

| Поле | Тип | Обязательный | Описание |
|---|---|---|---|
| `warehouseId` | int64 | **да** | ID склада, на котором проводится инвентаризация |
| `documentNo` | string | нет | Номер документа; если не указан — генерируется автоматически (`INV-...`) |
| `note` | string | нет | Примечание |
| `items` | array | **да** | Список товаров (минимум 1 позиция) |
| `items[].productId` | int64 | нет | ID товара из справочника |
| `items[].productName` | string | **да** | Название товара |
| `items[].expectedQty` | float64 | нет | Ожидаемое количество по системе (заполняется из остатков) |
| `items[].actualQty` | float64 | нет | Фактически пересчитанное количество |
| `items[].costPrice` | float64 | нет | Себестоимость единицы (для расчёта излишка/недостачи) |

---

### PUT `/warehouse/inventories/{id}/post`

Проводит документ инвентаризации. Статус меняется с `draft` → `posted`.

**Что происходит при проведении:**
- Для каждой позиции с `productId`: остаток на складе устанавливается равным `actualQty`
- Вычисляется `surplusDeficit = SUM((actualQty - expectedQty) × costPrice)`
  - Положительное значение — **излишек** (товара больше, чем ожидалось)
  - Отрицательное значение — **недостача** (товара меньше, чем ожидалось)

**Права:** `warehouse.inventory.post`

**Параметры пути:**

| Параметр | Тип | Описание |
|---|---|---|
| `id` | int64 | ID документа инвентаризации |

**Ошибки:**
- `409 Conflict` — документ уже проведён (`already posted`)
- `404 Not Found` — документ не найден

---

**Поля ответа (InventoryOrderResponse):**

| Поле | Тип | Описание |
|---|---|---|
| `id` | int64 | ID документа |
| `companyID` | int64 | ID компании |
| `warehouseId` | int64 | ID склада |
| `warehouseName` | string | Название склада |
| `documentNo` | string | Номер документа |
| `status` | string | `draft` / `posted` / `cancelled` |
| `surplusDeficit` | float64 | Сумма излишка/недостачи (после проведения) |
| `note` | string | Примечание |
| `createdBy` | int64 | ID пользователя, создавшего документ |
| `createdByName` | string | Имя пользователя (Создал) |
| `updatedBy` | int64 | ID пользователя, проводившего документ |
| `updatedByName` | string | Имя пользователя (Изменил) |
| `createdAt` | datetime | Дата создания |
| `updatedAt` | datetime | Дата последнего изменения |
| `items` | array | Позиции (возвращаются при создании/проведении) |

---

## 4. Перемещение товаров

Перемещение позволяет передать товары с одного склада на другой. При проведении документа остатки уменьшаются на складе-источнике и увеличиваются на складе-получателе.

### GET `/warehouse/transfers`

Список документов перемещения.

**Права:** `warehouse.transfer.read`

**Query-параметры:**

| Параметр | Тип | Обязательный | Описание |
|---|---|---|---|
| `from_warehouse_id` | int64 | нет | Фильтр по складу-источнику |
| `to_warehouse_id` | int64 | нет | Фильтр по складу-получателю |
| `status` | string | нет | `draft`, `posted`, `cancelled` |
| `date_from` | date | нет | Фильтр с даты (формат: `2024-01-01`) |
| `date_to` | date | нет | Фильтр по дату включительно |
| `limit` | int | нет | Кол-во записей |
| `offset` | int | нет | Смещение |

---

### POST `/warehouse/transfers`

Создаёт документ перемещения со статусом `draft`.

**Права:** `warehouse.transfer.create`

**Тело запроса:**
```json
{
  "fromWarehouseId": 1,
  "toWarehouseId": 2,
  "documentNo": "",
  "note": "Перемещение в филиал",
  "transferredAt": "2024-05-15T09:00:00Z",
  "items": [
    {
      "productId": 42,
      "productName": "Пикамилон 20мг №30тб",
      "quantity": 5.0,
      "costPrice": 21.00
    }
  ]
}
```

**Поля запроса:**

| Поле | Тип | Обязательный | Описание |
|---|---|---|---|
| `fromWarehouseId` | int64 | **да** | ID склада-источника |
| `toWarehouseId` | int64 | **да** | ID склада-получателя (не может совпадать с источником) |
| `documentNo` | string | нет | Номер документа; если не указан — генерируется автоматически (`TRF-...`) |
| `note` | string | нет | Примечание |
| `transferredAt` | datetime | нет | Дата перемещения; по умолчанию — текущее время |
| `items` | array | **да** | Список товаров (минимум 1 позиция) |
| `items[].productId` | int64 | нет | ID товара |
| `items[].productName` | string | **да** | Название товара |
| `items[].quantity` | float64 | **да** | Количество для перемещения (> 0) |
| `items[].costPrice` | float64 | нет | Себестоимость единицы |

---

### PUT `/warehouse/transfers/{id}/post`

Проводит документ перемещения. Статус меняется `draft` → `posted`.

**Что происходит при проведении:**
- Для каждой позиции: остаток уменьшается на складе-источнике на `quantity`
- Остаток увеличивается на складе-получателе на `quantity`
- Если товара на складе-источнике недостаточно — возвращается `400 Bad Request`

**Права:** `warehouse.transfer.post`

**Ошибки:**
- `400 Bad Request` — недостаточно товара на складе-источнике
- `409 Conflict` — документ уже проведён

**Поля ответа (TransferOrderResponse):**

| Поле | Тип | Описание |
|---|---|---|
| `id` | int64 | ID документа |
| `documentNo` | string | Номер документа |
| `fromWarehouseId` | int64 | ID склада-источника |
| `fromWarehouseName` | string | Название склада-источника (Со склада) |
| `toWarehouseId` | int64 | ID склада-получателя |
| `toWarehouseName` | string | Название склада-получателя (На склад) |
| `status` | string | `draft` / `posted` / `cancelled` |
| `note` | string | Примечание |
| `transferredAt` | datetime | Дата перемещения |
| `receivedAt` | datetime | Дата получения (может быть null) |
| `createdBy` | int64 | ID пользователя |
| `createdAt` | datetime | Дата создания |
| `updatedAt` | datetime | Дата изменения |
| `items` | array | Позиции документа |

---

## 5. Списание товаров

Списание позволяет удалить товары со склада (брак, истечение срока, порча и т.д.). При проведении документа остатки уменьшаются.

### GET `/warehouse/writeoffs`

Список документов списания.

**Права:** `warehouse.writeoff.read`

**Query-параметры:**

| Параметр | Тип | Обязательный | Описание |
|---|---|---|---|
| `warehouse_id` | int64 | нет | Фильтр по складу |
| `status` | string | нет | `draft`, `posted`, `cancelled` |
| `date_from` | date | нет | Фильтр с даты (формат: `2024-01-01`) |
| `date_to` | date | нет | Фильтр по дату включительно |
| `limit` | int | нет | Кол-во записей |
| `offset` | int | нет | Смещение |

---

### POST `/warehouse/writeoffs`

Создаёт документ списания со статусом `draft`. `totalAmount` вычисляется автоматически как `SUM(quantity × costPrice)`.

**Права:** `warehouse.writeoff.create`

**Тело запроса:**
```json
{
  "warehouseId": 1,
  "documentNo": "",
  "objectName": "Инвентарь №3",
  "counterpartyName": "",
  "note": "Истёк срок годности",
  "writtenOffAt": "2024-05-15T09:00:00Z",
  "items": [
    {
      "productId": 42,
      "productName": "Пикамилон 20мг №30тб",
      "quantity": 3.0,
      "costPrice": 21.00
    }
  ]
}
```

**Поля запроса:**

| Поле | Тип | Обязательный | Описание |
|---|---|---|---|
| `warehouseId` | int64 | **да** | ID склада |
| `documentNo` | string | нет | Номер документа; если не указан — генерируется автоматически (`WOF-...`) |
| `objectName` | string | нет | Объект/причина списания |
| `counterpartyName` | string | нет | Контрагент, связанный со списанием |
| `note` | string | нет | Примечание |
| `writtenOffAt` | datetime | нет | Дата списания; по умолчанию — текущее время |
| `items` | array | **да** | Список товаров (минимум 1 позиция) |
| `items[].productId` | int64 | нет | ID товара |
| `items[].productName` | string | **да** | Название товара |
| `items[].quantity` | float64 | **да** | Количество к списанию (> 0) |
| `items[].costPrice` | float64 | нет | Себестоимость единицы |

---

### PUT `/warehouse/writeoffs/{id}/post`

Проводит документ списания. Статус меняется `draft` → `posted`.

**Что происходит при проведении:**
- Для каждой позиции: остаток уменьшается на `quantity`
- Если товара на складе недостаточно — возвращается `400 Bad Request`

**Права:** `warehouse.writeoff.post`

**Поля ответа (WriteoffOrderResponse):**

| Поле | Тип | Описание |
|---|---|---|
| `id` | int64 | ID документа |
| `warehouseId` | int64 | ID склада |
| `warehouseName` | string | Название склада |
| `documentNo` | string | Номер документа |
| `status` | string | `draft` / `posted` / `cancelled` |
| `totalAmount` | float64 | Итоговая сумма = `SUM(quantity × costPrice)` |
| `objectName` | string | Объект списания |
| `counterpartyName` | string | Контрагент |
| `note` | string | Примечание |
| `writtenOffAt` | datetime | Дата списания |
| `createdBy` | int64 | ID пользователя |
| `createdByName` | string | Имя пользователя (Пользователь в UI) |
| `createdAt` | datetime | Дата создания |
| `updatedAt` | datetime | Дата изменения |
| `items` | array | Позиции документа |

---

## 6. Коды статусов HTTP

| Код | Значение | Когда возникает |
|---|---|---|
| `200` | Успех | Запрос выполнен |
| `400` | Неверный запрос | Не переданы обязательные поля, неверные данные, или недостаточно остатков |
| `401` | Не авторизован | Отсутствует или просрочен токен |
| `403` | Запрещено | Нет необходимых прав |
| `404` | Не найдено | Документ с указанным ID не существует |
| `409` | Конфликт | Дублирующий номер документа, или документ уже проведён |
| `500` | Ошибка сервера | Внутренняя ошибка |

---

## 7. Статусы документов

| Статус | Описание | Переход |
|---|---|---|
| `draft` | Черновик. Остатки не изменяются | Создаётся по умолчанию |
| `posted` | Проведён. Остатки обновлены | `draft` → `posted` через `PUT /{id}/post` |
| `cancelled` | Отменён | Планируется в следующей версии |

> **Важно:** Проведение документа необратимо через API. После перевода в `posted` документ нельзя снова сделать `draft`.

---

## Порядок работы (типичный сценарий)

### Инвентаризация
```
1. GET /warehouse/stock — посмотреть текущие остатки
2. POST /warehouse/inventories — создать документ (status=draft)
   → передать expectedQty из текущих остатков, actualQty — результат пересчёта
3. PUT /warehouse/inventories/{id}/post — провести
   → система обновит остатки до actualQty
```

### Перемещение
```
1. GET /warehouse/warehouses — получить список складов
2. POST /warehouse/transfers — создать документ (status=draft)
3. PUT /warehouse/transfers/{id}/post — провести
   → остатки перемещены между складами
```

### Списание
```
1. GET /warehouse/stock — проверить наличие товара
2. POST /warehouse/writeoffs — создать документ (status=draft)
3. PUT /warehouse/writeoffs/{id}/post — провести
   → остатки уменьшены
```