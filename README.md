# Notification Service
REST API сервис, написанный на Go, для отправки мгновенных и отложенных уведомлений по электронной почте.

---


## API Endpoints

### 1. Мгновенная отправка письма

\
**Описание:**

```text
Осуществляет мгновенную отправку письма, используя Simple Male Transfer Protocol (SMTP),
на указанный адрес электронной почты, с заданным заголовком и текстом письма.
После успешной отправки и последующего сохранения письма в PostgreSQL, клиенту выдается уникальный ID письма.
```
\
**Endpoint:**  
`POST: /send-notification`

\
**Request Body (JSON):**

```json
{
  "to": "youremail@gmail.com",
  "subject": "your subject",
  "message": "your message"
}
```

\
**Response success (JSON):**

```json
{"message":"Successfully sent notification","id":1}
```

---


### 2. Отправка отложенного письма

\
**Описание**

```text
Осуществляет отправку письма к заданному времени. После проверки корректности тела запроса,
письмо сохраняется в PostgreSQL и выдется уникальный ID, затем письмо попадает в Redis,
после фоновый Worker по заданному времени интервала проверяет базу данных и если находит письмо, 
время отправки которого пришло, начинает отправку письма
```

\
**Endpoint:**  
`POST: /send-notification-via-time`

\
**Request Body (JSON):**

```json
{
  "time": "2025-07-13 11:58:00",
  "to": "yourmail@gmail.com",
  "subject": "something subject",
  "message": "something message"
}
```

\
**Response success (JSON):**

```json
{"message":"Successfully saved your mail","id":2}
```

---


### 3. Просмотр истории отправок

\
**Описание:**
```text
Осуществляет выдачу клиенту отправленных и сохраненных ранее писем, используя одного из трех типов Query Parameters:
по уникальному ID, по адресу электронной почты получателя и полная выдача всех имеющихся сохраненных писем
```

\
**Endpoint:**  
`GET: /list`

\
**Query:**

```text
  /list?by=id&id=1
  
  /list?by=email&email=something@gmail.com
  
  /list?by=all
```

\
**Response success (JSON):**

```json
[{"type":"instantSending","to":"youremail@gmail.com","subject":"your subject","message":"your message"}]
```

---


## Примеры cURL

\
**Отправка мгновенного письма**

```bash
curl -X POST http://localhost:8080/send-notification \                       
-H "Content-Type: application/json" \
-d '{
  "to":"yourmail@gmail.com",
  "subject":"subject",
  "message":"message"
  }'
```

\
**Отправка отложенного письма**

```bash
curl -X POST http://localhost:8080/send-notification-via-time \                       
-H "Content-Type: application/json" \
-d '{
  "time": "2025-07-13 11:58:00",
  "to":"yourmail@gmail.com",
  "subject":"subject",
  "message":"message"
  }'
```

\
**Выдача сохраненных писем по ID**

```bash
curl -X GET http://localhost:8080/list?by=id&id=1
```

\
**Выдача сохраненных писем по адресу электронной почты получателя**

```bash
curl -X GET http://localhost:8080/list?by=email&email=something@gmail.com
```

\
**Выдача всех сохраненных писем**

```bash
curl -X GET http://localhost:8080/list?by=all
```

---


## Запуск приложения

```bash
make all
```

---


## Тестирование

```text
Были реализованы unit и integration тесты для абсолютно каждого пакета (покрытие тестами по пакетам более 75%)
```

## Запуск всех тестов в проекте

```bash
go test ./...  
```

---


## Используемые технологии

```text
- SMTP (net/smtp)
- Exponential Retry при ошибках отправки письма
- Redis (Redis Cluster)
- PostgreSQL (вместе с миграциями)
- Фоновый Worker который с указанным интервалом асинхронно ходит в Redis и ищет записи
- chi router
- Docker, Docker Compose
- Makefile
- bash scripts
- Prometheus
- Grafana
- GitHub Actions (после каждого git push запускает все тесты в проекте)
- Graceful Shutdown. Как на стороне HTTP-сервера, так и на стороне клиента
- Работа с Goroutine
- Работа с Context
- Повсеместные context timeout
- Документация каждого пакета (также для неэкспортируемых сущностей)
- Unit-тесты
- Integration-тесты
- MailHog
- testcontainers-go
```

---
