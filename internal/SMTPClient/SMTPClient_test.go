package SMTPClient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"notification/internal/monitoring"
)

func TestSendEmail(t *testing.T) {
	host, port, mailHogPort, url := upMailHog(context.Background(), t)

	tests := []struct {
		name      string
		ctx       context.Context
		from      string
		wantFrom  string
		email     *EmailMessage
		wantEmail *EmailMessage
		wantErr   error
	}{
		{
			name:     "success send",
			ctx:      context.Background(),
			from:     "something@gmail.com",
			wantFrom: "something@gmail.com",
			email: &EmailMessage{
				To:      "daanisimov04@gmail.com",
				Subject: "hi",
				Message: "hello from go test",
			},
			wantEmail: &EmailMessage{
				To:      "daanisimov04@gmail.com",
				Subject: "hi",
				Message: "hello from go test",
			},
			wantErr: nil,
		},
		{
			name:     "no valid sender address",
			ctx:      context.Background(),
			from:     "something",
			wantFrom: "something@gmail.com",
			email: &EmailMessage{
				To:      "daanisimov04@gmail.com",
				Subject: "hi",
				Message: "hello from go test",
			},
			wantEmail: nil,
			wantErr:   ErrNoValidSenderAddress,
		},
		{
			name: "context canceled before send",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			from:     "something@gmail.com",
			wantFrom: "something@gmail.com",
			email: &EmailMessage{
				To:      "daanisimov04@gmail.com",
				Subject: "hi",
				Message: "hello from go test",
			},
			wantEmail: nil,
			wantErr:   ErrContextCanceledBeforeSending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New(&Config{
				SenderEmail: tt.from,
				SMTPHost:    host,
				SMTPPort:    port,
			}, monitoring.NewNop(), zap.NewNop())

			err := srv.SendEmail(tt.ctx, *tt.email)
			assert.ErrorIs(t, err, tt.wantErr)

			if tt.wantEmail != nil {
				time.Sleep(1 * time.Second)

				gotFrom, gotTo, gotSubject, gotMessage := parseMailHogResponse(url, t)

				assert.Equal(t, gotFrom, tt.wantFrom)
				assert.Equal(t, gotTo, tt.wantEmail.To)
				assert.Equal(t, gotSubject, tt.wantEmail.Subject)
				assert.Equal(t, gotMessage, tt.wantEmail.Message)
			}

			cleanMailHog(mailHogPort, t)
		})
	}
}

func TestServerUnreachable(t *testing.T) {
	srv := New(&Config{
		SenderEmail:     "something@gmail.com",
		SMTPHost:        "localhost",
		SMTPPort:        9999,
		SenderPassword:  "invalid",
		MaxRetries:      2,
		BasicRetryPause: 1,
	}, monitoring.NewNop(), zap.NewNop())

	t.Run("smtp server unreachable", func(t *testing.T) {

		err := srv.SendEmail(context.Background(), EmailMessage{
			To:      "daanisimov04@gmail.com",
			Subject: "hi",
			Message: "hello from go test",
		})

		assert.Error(t, err)

		assert.Contains(t, err.Error(), "sendWithRetry: all attempts to send message failed")
	})

}

func TestCreatePause(t *testing.T) {
	srv := New(&Config{BasicRetryPause: 3 * time.Second}, nil, nil)

	tests := []struct {
		name       string
		i          int
		basicPause time.Duration
		want       time.Duration
	}{
		{
			name:       "i = 1",
			i:          1,
			basicPause: 3 * time.Second,
			want:       3 * time.Second,
		},
		{
			name:       "i = 2",
			i:          2,
			basicPause: 3 * time.Second,
			want:       6 * time.Second,
		},
		{
			name:       "i = 3",
			i:          3,
			basicPause: 3 * time.Second,
			want:       12 * time.Second,
		},
		{
			name:       "i = 4",
			i:          4,
			basicPause: 3 * time.Second,
			want:       24 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := srv.CreatePause(tt.i)
			assert.Equal(t, tt.want, got)
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

func upMailHog(ctx context.Context, t *testing.T) (string, int, string, string) {
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
		t.Fatalf("cannot get port: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("cannot get host: %v", err)
	}

	mailHogPort, err := container.MappedPort(ctx, "8025/tcp")
	if err != nil {
		t.Fatalf("cannot get mailHog port: %v", err)
	}

	url := fmt.Sprintf("http://localhost:%s/api/v2/messages", mailHogPort.Port())

	return host, port.Int(), mailHogPort.Port(), url
}

func cleanMailHog(port string, t *testing.T) {
	t.Helper()

	url := fmt.Sprintf("http://localhost:%s/api/v1/messages", port)
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
