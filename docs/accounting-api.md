# Бухгалтерия — API документация для фронтенда

## Базовый URL

```
/api/v1
```

## Авторизация

Все запросы требуют заголовок:

```
Authorization: Bearer <access_token>
```

Для мультитенантности передавать заголовок:

```
X-Company-ID: <company_id>
```

---

## Кнопка «Ещё» — логика для каждого формата

Каждый раздел с кнопкой «Ещё» имеет эндпоинт `/export` с query-параметром `?format=`:

| Кнопка | `format` | Действие на фронте |
|--------|----------|--------------------|
| **Excel** | `excel` | Скачать `.xlsx` файл через `<a download>` или Blob |
| **PDF** | `html` | Открыть в новой вкладке → браузер → «Сохранить как PDF» |
| **Печать** | `html` | Открыть в новой вкладке → `window.print()` |
| **Скопировать** | `tsv` | Получить текст → `navigator.clipboard.writeText(text)` |

### Пример: кнопка Excel

```javascript
async function exportExcel(url, filename) {
  const token = getAccessToken();
  const res = await fetch(url + '?format=excel', {
    headers: { Authorization: `Bearer ${token}` }
  });
  const blob = await res.blob();
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
}

// Использование
exportExcel('/api/v1/accounting/payment-documents/export', 'payment-docs.xlsx');
```

### Пример: кнопка PDF / Печать

```javascript
async function exportHTML(url) {
  const token = getAccessToken();
  const res = await fetch(url + '?format=html', {
    headers: { Authorization: `Bearer ${token}` }
  });
  const html = await res.text();
  const win = window.open('', '_blank');
  win.document.write(html);
  win.document.close();
  win.print(); // убрать эту строку если нужен только просмотр
}
```

### Пример: кнопка Скопировать

```javascript
async function exportCopy(url) {
  const token = getAccessToken();
  const res = await fetch(url + '?format=tsv', {
    headers: { Authorization: `Bearer ${token}` }
  });
  const text = await res.text();
  await navigator.clipboard.writeText(text);
  // показать toast "Скопировано"
}
```

---

## 1. Платежные документы

**Страница:** `Бухгалтерия / Платежные документы`  
**Кнопка «Ещё»:** ✅ Excel, PDF, Печать, Скопировать

### Список

```
GET /api/v1/accounting/payment-documents
```

**Query params:**

| Параметр | Тип | Описание |
|----------|-----|----------|
| `company_id` | int | ID компании (подставляется из scope) |
| `document_no` | string | Поиск по номеру документа |
| `doc_type` | string | Тип документа |
| `from` | date `YYYY-MM-DD` | Начало периода |
| `to` | date `YYYY-MM-DD` | Конец периода |
| `limit` | int | Кол-во (по умолчанию 20) |
| `offset` | int | Смещение (по умолчанию 0) |

**Ответ:**

```json
{
  "code": 200,
  "message": "success",
  "payload": [
    {
      "id": 1,
      "companyId": 1,
      "documentNo": "DOC-001",
      "docDate": "2026-06-10T00:00:00Z",
      "docType": "Приходный кассовый ордер",
      "debitAccount": "Касса",
      "creditAccount": "Расчётный счёт",
      "income": 5000.00,
      "expense": 0,
      "category": "Продажи",
      "note": "",
      "organization": "ООО Нурмед",
      "createdBy": 1,
      "createdAt": "2026-06-10T12:00:00Z",
      "updatedAt": "2026-06-10T12:00:00Z"
    }
  ]
}
```

### Создать

```
POST /api/v1/accounting/payment-documents
Content-Type: application/json

{
  "docType": "Расходный кассовый ордер",
  "docDate": "2026-06-10T00:00:00Z",
  "debitAccount": "Расходы",
  "creditAccount": "Касса",
  "expense": 1500.00,
  "category": "Аренда",
  "note": "Оплата аренды офиса",
  "organization": "ИП Иванов"
}
```

### Экспорт (кнопка «Ещё»)

```
GET /api/v1/accounting/payment-documents/export?format=excel&company_id=1&from=2026-01-01&to=2026-12-31
GET /api/v1/accounting/payment-documents/export?format=html&company_id=1
GET /api/v1/accounting/payment-documents/export?format=tsv&company_id=1
```

---

## 2. Наличность в кассе

**Страница:** `Бухгалтерия / Наличность в кассе`  
**Кнопка «Ещё»:** ✅ Excel, PDF, Печать, Скопировать

### Список (баланс на дату)

```
GET /api/v1/accounting/cash-balance
```

**Query params:**

| Параметр | Тип | Описание |
|----------|-----|----------|
| `company_id` | int | ID компании |
| `date` | date `YYYY-MM-DD` | Дата расчёта (по умолчанию сегодня) |

**Ответ:**

```json
{
  "code": 200,
  "message": "success",
  "payload": [
    {
      "id": 1,
      "name": "Алиф",
      "currency": "TJS",
      "balance": 3990.93,
      "balanceInTJS": 3990.93
    },
    {
      "id": 2,
      "name": "Эсхата",
      "currency": "TJS",
      "balance": 10900.58,
      "balanceInTJS": 10900.58
    }
  ]
}
```

### Экспорт (кнопка «Ещё»)

```
GET /api/v1/accounting/cash-balance/export?format=excel&company_id=1&date=2026-06-10
```

---

## 3. Выписка по счёту

**Страница:** `Бухгалтерия / Выписка по счёту`  
**Кнопка «Ещё»:** ✅ Excel, PDF, Печать, Скопировать

### Список

```
GET /api/v1/accounting/account-statement
```

**Query params:**

| Параметр | Тип | Описание |
|----------|-----|----------|
| `company_id` | int | ID компании |
| `cash_register_id` | int | ID кассы (обязательно для фильтрации) |
| `from` | date | Начало периода |
| `to` | date | Конец периода |
| `limit` | int | 20 |
| `offset` | int | 0 |

**Ответ:**

```json
{
  "code": 200,
  "message": "success",
  "payload": [
    {
      "id": 1,
      "docDate": "2026-06-01T09:00:00Z",
      "docNo": "PKO-001",
      "docType": "Приходный ордер",
      "openingBalance": 0,
      "income": 5000.00,
      "expense": 0,
      "closingBalance": 5000.00,
      "category": "Продажи",
      "note": "",
      "createdBy": 1
    }
  ]
}
```

### Экспорт (кнопка «Ещё»)

```
GET /api/v1/accounting/account-statement/export?format=excel&company_id=1&cash_register_id=1&from=2026-06-01&to=2026-06-30
```

---

## 4. Баланс контрагентов

**Страница:** `Бухгалтерия / Баланс контрагентов`  
**Кнопка «Ещё»:** ✅ Excel, PDF, Печать, Скопировать

### Список

```
GET /api/v1/accounting/counterparty-balance
```

**Query params:**

| Параметр | Тип | Описание |
|----------|-----|----------|
| `company_id` | int | ID компании |
| `group_type` | string | `Клиент` или `Поставщики` (пусто = все) |
| `search` | string | Поиск по имени |
| `limit` | int | 20 |
| `offset` | int | 0 |

> Дебиторка/Кредиторка определяется на фронте по знаку `amount`:
> - `amount > 0` → задолжали нам (дебиторка)
> - `amount < 0` → мы должны (кредиторка)

**Ответ:**

```json
{
  "code": 200,
  "message": "success",
  "payload": [
    {
      "id": 1,
      "no": 1,
      "groupType": "Клиент",
      "name": "Sobirov fakhridin",
      "phone": "+888550354",
      "region": "",
      "currency": "TJS",
      "amount": -20.00,
      "amountInTJS": -20.00
    }
  ]
}
```

### Экспорт (кнопка «Ещё»)

```
GET /api/v1/accounting/counterparty-balance/export?format=excel&company_id=1
GET /api/v1/accounting/counterparty-balance/export?format=excel&company_id=1&group_type=Клиент
```

---

## 5. Управление долгами

**Страница:** `Бухгалтерия / Управление долгами`  
**Кнопка «Ещё»:** ✅ Excel, PDF, Печать, Скопировать

### Список

```
GET /api/v1/accounting/debt-records
```

**Query params:**

| Параметр | Тип | Описание |
|----------|-----|----------|
| `company_id` | int | ID компании |
| `status` | string | `active` или `closed` |
| `search` | string | Поиск по имени клиента |
| `limit` | int | 20 |
| `offset` | int | 0 |

**Ответ:**

```json
{
  "code": 200,
  "message": "success",
  "payload": [
    {
      "id": 1,
      "companyId": 1,
      "clientName": "Иванов Иван",
      "phone": "+992900123456",
      "period": "12 месяцев",
      "startDate": "2026-01-01T00:00:00Z",
      "nextPaymentDate": "2026-07-01T00:00:00Z",
      "lastPaymentDate": "2026-06-01T00:00:00Z",
      "balance": 15000.00,
      "clientText": "Уважаемый клиент, напоминаем об оплате",
      "adminText": "Напомнить за 3 дня до срока",
      "note": "",
      "channels": "SMS, WhatsApp",
      "status": "active",
      "createdAt": "2026-01-01T00:00:00Z",
      "updatedAt": "2026-06-01T00:00:00Z"
    }
  ]
}
```

### Создать

```
POST /api/v1/accounting/debt-records
Content-Type: application/json

{
  "clientName": "Иванов Иван",
  "phone": "+992900123456",
  "period": "12 месяцев",
  "startDate": "2026-01-01T00:00:00Z",
  "nextPaymentDate": "2026-07-01T00:00:00Z",
  "balance": 15000.00,
  "channels": "SMS",
  "status": "active"
}
```

### Экспорт (кнопка «Ещё»)

```
GET /api/v1/accounting/debt-records/export?format=excel&company_id=1&status=active
```

---

## 6. Прайс-лист

**Страница:** `Бухгалтерия / Прайс-лист`  
**Кнопка «Ещё»:** ❌ (нет кнопки «Ещё», только Создать)

### Список

```
GET /api/v1/accounting/price-lists?company_id=1
```

**Ответ:**

```json
{
  "code": 200,
  "message": "success",
  "payload": [
    {
      "id": 1,
      "companyId": 1,
      "no": 1,
      "name": "Основной прайс-лист номенклатуры",
      "note": "",
      "priceListType": "nomenclature",
      "createdAt": "2026-01-01T00:00:00Z",
      "updatedAt": "2026-01-01T00:00:00Z"
    }
  ]
}
```

### Создать

```
POST /api/v1/accounting/price-lists
Content-Type: application/json

{
  "name": "Прайс-лист услуг",
  "priceListType": "service"
}
```

`priceListType`: `nomenclature` | `service`

---

## 7. Курсы валют

**Страница:** `Бухгалтерия / Курсы валют`  
**Кнопка «Ещё»:** ❌ (нет кнопки «Ещё», только Добавить)

### Список

```
GET /api/v1/accounting/currency-rates?company_id=1
```

**Ответ:**

```json
{
  "code": 200,
  "message": "success",
  "payload": [
    {
      "id": 1,
      "companyId": 1,
      "currency": "USD",
      "rate": 10.95,
      "rateDate": "2026-06-10T00:00:00Z",
      "createdBy": 1,
      "createdAt": "2026-06-10T09:00:00Z"
    }
  ]
}
```

### Добавить

```
POST /api/v1/accounting/currency-rates
Content-Type: application/json

{
  "currency": "USD",
  "rate": 10.95,
  "rateDate": "2026-06-10T00:00:00Z"
}
```

---

## Коды ошибок

| HTTP | Описание |
|------|----------|
| 200 | Успех |
| 400 | Неверный запрос (невалидные поля) |
| 401 | Не авторизован (нет или неверный токен) |
| 403 | Нет прав доступа |
| 500 | Внутренняя ошибка сервера |

**Формат ошибки:**

```json
{
  "code": 400,
  "message": "bad request",
  "payload": null
}
```

---

## Разрешения (permissions) для RBAC

| Страница | Чтение | Создание |
|----------|--------|----------|
| Платежные документы | `accounting.payment_doc.read` | `accounting.payment_doc.create` |
| Наличность в кассе | `accounting.cash.read` | — |
| Выписка по счёту | `accounting.statement.read` | — |
| Баланс контрагентов | `accounting.counterparty.read` | — |
| Управление долгами | `accounting.debt.read` | `accounting.debt.create` |
| Прайс-лист | `accounting.pricelist.read` | `accounting.pricelist.create` |
| Курсы валют | `accounting.currency.read` | `accounting.currency.create` |
