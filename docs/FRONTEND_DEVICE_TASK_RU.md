# Задача для фронтенда: Verify Device + Email Code

## Цель
Реализовать вход так, чтобы:
- новое устройство: пользователь подтверждает устройство через email-код;
- доверенное устройство: сразу открывается экран логин/пароль.

## Что уже делает backend
- `POST /api/v1/auth/check-device`:
  - `status = "verified"` -> устройство доверенное;
  - `status = "verification_required"` -> нужно подтверждение.
- `POST /api/v1/auth/verify-device` (2 шага):
  1) без `verificationCode` -> backend отправляет код на `identifier` (сейчас логируется);
  2) с `verificationCode` -> backend подтверждает устройство и ставит `trusted_device` cookie.
- `POST /api/v1/auth/login` -> обычный логин по `identifier + password`.

## Важно
Во всех auth/device запросах обязательно:
- `credentials: 'include'`

Иначе backend не получит `pending_device`/`trusted_device` cookie.

## Экранный flow
1. Открытие auth-страницы
   - вызвать `check-device` с `deviceInfo`.
2. Если `verified`
   - показать экран `логин + пароль`.
3. Если `verification_required`
   - показать экран `email`.
4. На отправку email
   - вызвать `verify-device` с `{ identifier, deviceInfo }`.
   - показать экран ввода кода.
5. На отправку кода
   - вызвать `verify-device` с `{ identifier, verificationCode, deviceInfo }`.
   - при успехе показать экран `логин + пароль`.
6. Выполнить `login`.

## Минимальные API функции (пример)
```js
const API = '/api/v1/auth';

export async function checkDevice(deviceInfo) {
  const res = await fetch(`${API}/check-device`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ deviceInfo })
  });
  return res.json();
}

export async function requestDeviceCode(identifier, deviceInfo) {
  const res = await fetch(`${API}/verify-device`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ identifier, deviceInfo })
  });
  return res.json();
}

export async function confirmDeviceCode(identifier, verificationCode, deviceInfo) {
  const res = await fetch(`${API}/verify-device`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ identifier, verificationCode, deviceInfo })
  });
  return res.json();
}

export async function login(identifier, password) {
  const res = await fetch(`${API}/login`, {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ identifier, password })
  });
  return res.json();
}
```

## Структура UI состояний
- `loading`
- `email`
- `code`
- `login`

Рекомендуемый state:
- `identifier`
- `verificationCode`
- `deviceInfo`
- `error`
- `isSubmitting`

## DeviceInfo
Передавать стабильно один и тот же набор полей:
- `browserName`
- `browserVersion`
- `osName`
- `osVersion`
- `deviceName`

## Ошибки для UX
- `401` при неверном коде/просроченном pending -> показать "Код неверный или истек".
- `403` при mismatch fingerprint -> показать "Устройство не совпадает".
- `500` -> общий fallback.

## Definition of Done (DoD)
- [ ] На доверенном устройстве открывается сразу экран логина.
- [ ] На новом устройстве показывается экран email -> код -> логин.
- [ ] При успешном вводе кода повторный вход на том же устройстве не требует email/кода.
- [ ] Все запросы используют `credentials: 'include'`.
- [ ] UX покрывает ошибки 401/403/500.

