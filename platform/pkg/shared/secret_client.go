package shared

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

var (
	ErrEmptyPayload   = errors.New("agent returned empty payload")
	ErrCommandTooLong = errors.New("command payload exceeds maximum allowed size")
)

type Config struct {
	SocketPath    string
	TargetService string
	Timeout       time.Duration
}

type PostgresCredentials struct {
	Username string `msgpack:"username" json:"username"`
	Password []byte `msgpack:"password" json:"password"`
}

type RedisCredentials struct {
	Username string `msgpack:"username,omitempty" json:"username,omitempty"`
	Password []byte `msgpack:"password" json:"password"`
}

type MinioCredentials struct {
	AccessKey string `msgpack:"access_key" json:"access_key"`
	SecretKey []byte `msgpack:"secret_key" json:"secret_key"`
}

type FullBootstrapResponse struct {
	Postgres *PostgresCredentials `msgpack:"postgres,omitempty" json:"postgres,omitempty"`
	Redis    *RedisCredentials    `msgpack:"redis,omitempty" json:"redis,omitempty"`
	Minio    *MinioCredentials    `msgpack:"minio,omitempty" json:"minio,omitempty"`
}

// Zeroize czyszczenie wrażliwych bajtów z pamięci RAM w miejscu
func Zeroize(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// BootstrapApp pobiera komplet sekretów za pomocą protokołu binarnego MessagePack przez UDS.
// Zwraca wskaźnik na strukturę oraz funkcję cleanup(), którą wywołujesz w `defer`.
func BootstrapApp(ctx context.Context, cfg Config, requiredServices []string) (*FullBootstrapResponse, func(), error) {
	reqMap := map[string]any{
		"target_service": cfg.TargetService,
		"services":       requiredServices,
	}

	reqBytes, err := msgpack.Marshal(reqMap)
	if err != nil {
		return nil, func() {}, fmt.Errorf("błąd binarnej serializacji żądania bootstrap: %w", err)
	}

	cmdPayload := append([]byte("BOOTSTRAP "), reqBytes...)

	rawResp, err := ExecCommand(ctx, cfg.SocketPath, cfg.Timeout, cmdPayload)
	if err != nil {
		return nil, func() {}, fmt.Errorf("bootstrap nie powiódł się: %w", err)
	}

	var response FullBootstrapResponse
	if err := msgpack.Unmarshal(rawResp, &response); err != nil {
		Zeroize(rawResp)
		return nil, func() {}, fmt.Errorf("błąd deserializacji binarnej response: %w", err)
	}

	cleanup := func() {
		Zeroize(rawResp)
		if response.Postgres != nil {
			Zeroize(response.Postgres.Password)
		}
		if response.Redis != nil {
			Zeroize(response.Redis.Password)
		}
		if response.Minio != nil {
			Zeroize(response.Minio.SecretKey)
		}
	}

	return &response, cleanup, nil
}

// RefreshPostgres pobiera odświeżone dane wyłącznie dla bazy danych Postgres.
func RefreshPostgres(ctx context.Context, cfg Config) (*PostgresCredentials, func(), error) {
	cmdPayload := fmt.Appendf(nil, "REFRESH %s postgres", cfg.TargetService)

	rawResp, err := ExecCommand(ctx, cfg.SocketPath, cfg.Timeout, cmdPayload)
	if err != nil {
		return nil, func() {}, fmt.Errorf("rotacja poświadczeń DB nie powiodła się: %w", err)
	}

	var creds PostgresCredentials
	if err := msgpack.Unmarshal(rawResp, &creds); err != nil {
		Zeroize(rawResp)
		return nil, func() {}, fmt.Errorf("błąd deserializacji postgres creds: %w", err)
	}

	cleanup := func() {
		Zeroize(rawResp)
		Zeroize(creds.Password)
	}

	return &creds, cleanup, nil
}

// ExecCommand realizuje binarny protokół UDS IPC w standardzie Length-Delimited:
// Wysyłanie: [4 bajty BigEndian u32 długość][payload polecenia]
// Odpowiedź:  [4 bajty BigEndian u32 długość][payload odpowiedzi]
func ExecCommand(ctx context.Context, socketPath string, timeout time.Duration, commandPayload []byte) ([]byte, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("błąd połączenia z UDS: %w", err)
	}
	defer conn.Close()

	if timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(timeout))
	}

	reqLen := uint32(len(commandPayload))
	if err := binary.Write(conn, binary.BigEndian, reqLen); err != nil {
		return nil, fmt.Errorf("błąd zapisu nagłówka długości: %w", err)
	}

	if _, err := conn.Write(commandPayload); err != nil {
		return nil, fmt.Errorf("błąd zapisu payloadu polecenia: %w", err)
	}

	var respLen uint32
	if err := binary.Read(conn, binary.BigEndian, &respLen); err != nil {
		return nil, fmt.Errorf("błąd odczytu długości ramki odpowiedzi: %w", err)
	}

	if respLen == 0 {
		return nil, ErrEmptyPayload
	}

	respBuf := make([]byte, respLen)
	if _, err := io.ReadFull(conn, respBuf); err != nil {
		Zeroize(respBuf)
		return nil, fmt.Errorf("błąd odczytu bajtów payloadu z UDS: %w", err)
	}

	return respBuf, nil
}
