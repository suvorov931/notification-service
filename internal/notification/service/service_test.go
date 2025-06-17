package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

// TODO: новые тестовые кейсы
// TODO: обработка всех ошибок
// TODO: добавить кастомных внятных ошибок и проверить их
// TODO: поддержка рюка

// TODO: тесты для sendWithRetry

func TestSendMessage(t *testing.T) {
	ctx := context.Background()

	host, port, url := upMailHog(ctx, t)

	tests := []struct {
		name      string
		from      string
		wantFrom  string
		email     *Email
		wantEmail *Email
		wantErr   error
	}{
		{
			name:     "successful send",
			from:     "something@gmail.com",
			wantFrom: "something@gmail.com",
			email: &Email{
				To:      "daanisimov04@gmail.com",
				Subject: "hi",
				Message: "hello from go test",
			},
			wantEmail: &Email{
				To:      "daanisimov04@gmail.com",
				Subject: "hi",
				Message: "hello from go test",
			},
			wantErr: nil,
		},
	}

	t.Run("smtp server unreachable", func(t *testing.T) {
		srv := New(&MailSender{
			SenderEmail:     "something@gmail.com",
			SMTPHost:        "localhost",
			SMTPPort:        9999,
			SenderPassword:  "invalid",
			MaxRetries:      2,
			BasicRetryPause: 1,
		}, zap.NewNop())

		err := srv.SendMessage(ctx, Email{
			To:      "daanisimov04@gmail.com",
			Subject: "hi",
			Message: "hello from go test",
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SendMessage: all attempts to send message failed")
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New(&MailSender{
				SenderEmail: tt.from,
				SMTPHost:    host,
				SMTPPort:    port,
			}, zap.NewNop())
			err := srv.SendMessage(ctx, *tt.email)
			time.Sleep(time.Second)

			if err != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("SendMessage() error = %v, wantErr = %v", err, tt.wantErr)
				}
				t.SkipNow()
				return
			}

			gotFrom, gotTo, gotSubject, gotMessage := parseMailHogResponse(url, t)

			assert.Equal(t, gotFrom, tt.wantFrom)
			assert.Equal(t, gotTo, tt.wantEmail.To)
			assert.Equal(t, gotSubject, tt.wantEmail.Subject)
			assert.Equal(t, gotMessage, tt.wantEmail.Message)

			cleanMailHog(port, t)
		})
	}
}

type mailHogResponse struct {
	Total int `json:"total"`
	Items []struct {
		Content struct {
			Body    string              `json:"body"`
			Headers map[string][]string `json:"headers"`
		} `json:"content"`
	} `json:"items"`
}

func parseMailHogResponse(url string, t *testing.T) (string, string, string, string) {
	t.Helper()

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Cannot get response: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Cannot read body: %v", err)
	}

	got := mailHogResponse{}
	if err = json.Unmarshal(body, &got); err != nil {
		t.Fatalf("Cannot unmarshal body: %v", err)
	}

	from := got.Items[0].Content.Headers["From"][0]

	to := got.Items[0].Content.Headers["To"][0]

	subject := got.Items[0].Content.Headers["Subject"][0]

	message := got.Items[0].Content.Body

	return from, to, subject, message
}

func upMailHog(ctx context.Context, t *testing.T) (string, int, string) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Name:         "mailhog-for-tests",
		Image:        "mailhog/mailhog:v1.0.1",
		ExposedPorts: []string{"1025/tcp", "8025/tcp"},
		WaitingFor:   wait.ForListeningPort("1025/tcp").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            false,
	})
	if err != nil {
		t.Fatalf("Failed to start container %v", err)
	}

	port, err := container.MappedPort(ctx, "1025/tcp")
	if err != nil {
		t.Fatalf("cannot get mapped port: %v", err)
	}

	httpPort, err := container.MappedPort(ctx, "8025/tcp")
	if err != nil {
		t.Fatalf("cannot get http port: %v", err)
	}

	url := fmt.Sprintf("http://localhost:%s/api/v2/messages", httpPort.Port())

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("cannot get host: %v", err)
	}

	return host, port.Int(), url
}

func cleanMailHog(port int, t *testing.T) {
	t.Helper()

	url := fmt.Sprintf("http://localhost:%d/api/v1/messages", port)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("cleanMailHog: cannot create http request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("cleanMailHog: cannot execute http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cleanMailHog: status code not ok: %v", resp.StatusCode)
	}
}
