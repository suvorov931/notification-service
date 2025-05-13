# Notification Service
Микросервис REST API написанный на Go, для отправки уведомлений по электронной почте, с логикой повторных попыток (exponential retry) и плавным завершением работы (graceful shutdown).
Сервис разработан для командного проектного этапа, на курсе Yandex Lyceum "Веб-разработка на Go | Специализации Яндекс Лицея | Весна 24/25"



## API Endpoints

### 1. Отправка письма

**Endpoint:**  
`POST: /send-notification`

**Request Body (JSON):**

```json
{
  "to": "yourmail@gmail.com",
  "subject": "something subject",
  "message": "something message"
}
```

**Response (Success):**

```text
Message is correct,
Starting to send notification

Successfully sent notification
```

---

### 2. Отправка писем к определенному времени

### В разработке

---

## Примеры cURL

### Отправка письма

```bash
curl -X POST http://localhost:8080/send-notification \                            ─╯
-H "Content-Type: application/json" \
-d '{
    "to":"yourmail@gmail.com",
    "subject":"subject",
    "message":"message"
}'
```

---

## Запуск приложения

```bash
запуск на локальной машине:
go run cmd/main.go 

запуск в docker container:
docker compose up --build -d
```

---

## Тестирование
```text
На данном этапе были реализованы только unit тесты для кастомного json декодера,
который присылает клиенту осмысленные ошибки в случаях ошибочного парсинга json структуры.
```
```bash
# запуск всех тестов в проекте (необходимо использовать команду в корне проекта)
go test ./... -v   

# отдельный запуск тестов для json декодера (запускать из папки ./internal/notification/api/decoder)
go test  -v  
```

---

## Общая информация о проекте

```text
- chi router
- SMTP (net/smtp)
- Docker, Docker Compose
- Graceful Shutdown Как на стороне HTTP-сервера, так и при завершении SMTP-клиента
- Exponential Retry При ошибках отправки
- Unit-тесты
``` 

---

## В скором времени
```text
Будут добавлены unit тесты для каждого пакета и интеграционные тесты.
Будет добавлен второй метод сервиса - отправка писем к определенному времени.
Этот метод будет реализовывать Apache Kafka и Redis 
```