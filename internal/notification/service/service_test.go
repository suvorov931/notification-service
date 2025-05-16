package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"notification/internal/config"
)

// TODO: новые тестовые кейсы
// TODO: обработка всех ошибок
// TODO: тесты для sendWithRetry
// TODO: добавить кастомных внятных ошибок и проверить их

func TestSendMessage(t *testing.T) {
	ctx := context.Background()

	container := upMailHog(ctx, t)
	defer downMailHog(ctx, container, t)

	//stmpPort, err := container.MappedPort(ctx, "1025/tcp")
	//if err != nil {
	//	t.Errorf("cannot get  mapped port: %v", err)
	//}
	//
	//httpPort, err := container.MappedPort(ctx, "8025/tcp")
	//if err != nil {
	//	t.Errorf("cannot get http port: %v", err)
	//}
	//url := fmt.Sprintf("http://localhost:%s/api/v2/messages", httpPort.Port())
	//
	//port, err := strconv.Atoi(stmpPort.Port())
	//if err != nil {
	//	t.Errorf("cannot convert mapped port: %v", err)
	//}

	port, httpPort, url := getMailHogPorts(ctx, container, t)

	tests := []struct {
		name     string
		from     string
		wantFrom string
		mail     Mail
		wantMail Mail
		wantErr  error
	}{
		{
			name:     "successful send",
			from:     "something@gmail.com",
			wantFrom: "something@gmail.com",
			mail: Mail{
				To:      "daanisimov04@gmail.com",
				Subject: "hi",
				Message: "hello from go test",
			},
			wantMail: Mail{
				To:      "daanisimov04@gmail.com",
				Subject: "hi",
				Message: "hello from go test",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New(&config.CredentialsSender{
				SenderEmail: tt.from,
				SMTPHost:    "localhost",
				SMTPPort:    port,
			}, zap.NewNop())
			err := srv.SendMessage(ctx, tt.mail)
			time.Sleep(time.Second)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("SendMessage() error = %v, wantErr %v", err, tt.wantErr)
			}

			got := parseMailHogResponse(url, t)

			gotFrom := got.Items[0].Content.Headers["From"][0]
			if !reflect.DeepEqual(gotFrom, tt.wantFrom) {
				t.Errorf("SendMessage() gotFrom = %v, want %v", gotFrom, tt.wantFrom)
			}

			gotTo := got.Items[0].Content.Headers["To"][0]
			if !reflect.DeepEqual(gotTo, tt.wantMail.To) {
				t.Errorf("SendMessage() gotTo = %v, want %v", gotTo, tt.wantMail.To)
			}

			gotSubject := got.Items[0].Content.Headers["Subject"][0]
			if !reflect.DeepEqual(gotSubject, tt.wantMail.Subject) {
				t.Errorf("SendMessage() gotSubject = %v, want %v", gotSubject, tt.wantMail.Subject)
			}

			gotMessage := got.Items[0].Content.Body
			if !reflect.DeepEqual(gotMessage, tt.wantMail.Message) {
				t.Errorf("SendMessage() gotMessage = %v, want %v", gotMessage, tt.wantMail.Message)
			}

			cleanMailHog(httpPort, t)
		})
	}
}

func getMailHogPorts(ctx context.Context, container testcontainers.Container, t *testing.T) (int, string, string) {
	stmpPort, err := container.MappedPort(ctx, "1025/tcp")
	if err != nil {
		t.Errorf("cannot get  mapped port: %v", err)
	}

	httpPort, err := container.MappedPort(ctx, "8025/tcp")
	if err != nil {
		t.Errorf("cannot get http port: %v", err)
	}

	url := fmt.Sprintf("http://localhost:%s/api/v2/messages", httpPort.Port())

	port, err := strconv.Atoi(stmpPort.Port())
	if err != nil {
		t.Errorf("cannot convert mapped port: %v", err)
	}

	return port, httpPort.Port(), url
}

func parseMailHogResponse(url string, t *testing.T) mailHogResponse {
	resp, err := http.Get(url)
	if err != nil {
		t.Errorf("Cannot get response: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Cannot read body: %v", err)
	}

	got := mailHogResponse{}
	if err = json.Unmarshal(body, &got); err != nil {
		t.Errorf("Cannot unmarshal body: %v", err)
	}

	if got.Total == 0 {
		t.Fatalf("Expected 1 email in MailHog, got %d", got.Total)
	}

	return got
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

func upMailHog(ctx context.Context, t *testing.T) testcontainers.Container {
	t.Helper()

	if err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"); err != nil {
		t.Fatalf("cannot disable ryuk")
	}

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

	return container
}

func downMailHog(ctx context.Context, container testcontainers.Container, t *testing.T) {
	t.Helper()

	if err := container.Terminate(ctx); err != nil {
		t.Errorf("cannot terminate container: %v", err)
	}
}

func cleanMailHog(port string, t *testing.T) {
	t.Helper()

	url := fmt.Sprintf("http://localhost:%s/api/v1/messages", port)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Errorf("cleanMailHog: cannot create http request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("cleanMailHog: cannot execute http request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("cleanMailHog: status code not ok: %v", resp.StatusCode)
	}
}
