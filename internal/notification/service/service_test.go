package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

	c := config.Config{CredentialsSender: config.CredentialsSender{
		SenderEmail: "daa@gmail.com",
		SMTPHost:    "localhost",
		SMTPPort:    port,
	}}

	srv := New(&c, zap.NewNop())

	m := Mail{
		To:      "daanisimov04@gmail.com",
		Subject: "hi",
		Message: "hello from go test",
	}

	srv.SendMessage(ctx, m)

	time.Sleep(time.Second)
	httpPort, _ := container.MappedPort(ctx, "8025/tcp")
	url := fmt.Sprintf("http://localhost:%s/api/v2/messages", httpPort.Port())
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	r := response{}
	json.Unmarshal(body, &r)

	gotSubject := r.Items[0].Content.Headers["Subject"][0]
	gotMessage := r.Items[0].Content.Body
	fmt.Println(gotSubject, m.Subject)
	fmt.Println(gotMessage, m.Message)
	fmt.Println(gotSubject == m.Subject, gotMessage == m.Message)
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
