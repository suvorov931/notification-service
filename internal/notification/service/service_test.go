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
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"notification/internal/config"
)

func TestSendMessage(t *testing.T) {
	ctx := context.Background()

	container := upMailHog(ctx)
	defer downMailHog(ctx, container)

	stmpPort, _ := container.MappedPort(ctx, "1025/tcp")
	port, _ := strconv.Atoi(stmpPort.Port())

	c := config.CredentialsSender{
		SenderEmail: "daa@gmail.com",
		SMTPHost:    "localhost",
		SMTPPort:    port,
	}

	srv := New(&c, zap.NewNop())
	httpPort, _ := container.MappedPort(ctx, "8025/tcp")
	url := fmt.Sprintf("http://localhost:%s/api/v2/messages", httpPort.Port())

	tests := []struct {
		name    string
		mail    Mail
		want    Mail
		wantErr error
	}{
		{
			name: "successful send",
			mail: Mail{
				To:      "daanisimov04@gmail.com",
				Subject: "hi",
				Message: "hello from go test",
			},
			want: Mail{
				To:      "daanisimov04@gmail.com",
				Subject: "hi",
				Message: "hello from go test",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := srv.SendMessage(ctx, tt.mail)
			time.Sleep(time.Second)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("SendMessage() error = %v, wantErr %v", err, tt.wantErr)
			}

			resp, err := http.Get(url)
			if err != nil {
				t.Errorf("Cannot get response: %v", err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("Cannot read body: %v", err)
			}

			got := response{}
			if err = json.Unmarshal(body, &got); err != nil {
				t.Errorf("Cannot unmarshal body: %v", err)
			}

			gotSubject := got.Items[0].Content.Headers["Subject"][0]
			if !reflect.DeepEqual(gotSubject, tt.want.Subject) {
				t.Errorf("SendMessage() gotSubject = %v, want %v", gotSubject, tt.want.Subject)
			}

			gotMessage := got.Items[0].Content.Body
			if !reflect.DeepEqual(gotMessage, tt.want.Message) {
				t.Errorf("SendMessage() gotMessage = %v, want %v", gotMessage, tt.want.Message)
			}
		})
	}
}

type response struct {
	Total int `json:"total"`
	Items []struct {
		Content struct {
			Body    string              `json:"body"`
			Headers map[string][]string `json:"headers"`
		} `json:"content"`
	} `json:"items"`
}

func upMailHog(ctx context.Context) testcontainers.Container {
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

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
		log.Printf("Failed to start container %v", err)
	}

	fmt.Println(container.Ports(ctx))

	return container
}

func downMailHog(ctx context.Context, container testcontainers.Container) {
	if err := container.Terminate(ctx); err != nil {
		log.Printf("cannot terminate container: %v", err)
	}
}
