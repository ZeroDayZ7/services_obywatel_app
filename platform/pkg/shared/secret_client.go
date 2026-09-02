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

type RabbitMQCredentials struct {
	Username string `msgpack:"username" json:"username"`
	Password []byte `msgpack:"password" json:"password"`
}

type FullBootstrapResponse struct {
	Postgres *PostgresCredentials `msgpack:"postgres,omitempty" json:"postgres,omitempty"`
	Redis    *RedisCredentials    `msgpack:"redis,omitempty" json:"redis,omitempty"`
	Minio    *MinioCredentials    `msgpack:"minio,omitempty" json:"minio,omitempty"`
	RabbitMQ *RabbitMQCredentials `msgpack:"rabbitmq,omitempty" json:"rabbitmq,omitempty"`
}

// BootstrapApp pobiera komplet sekretów za pomocą protokołu binarnego MessagePack przez UDS.
func BootstrapApp(ctx context.Context, cfg Config, requiredServices []string) (*FullBootstrapResponse, func(), error) {
	log := GetLogger()

	reqMap := map[string]any{
		"target_service": cfg.TargetService,
		"services":       requiredServices,
	}

	reqBytes, err := msgpack.Marshal(reqMap)
	if err != nil {
		return nil, func() {}, fmt.Errorf("błąd binarnej serializacji żądania bootstrap: %w", err)
	}

	cmdPayload := append([]byte("BOOTSTRAP "), reqBytes...)

	log.InfoMap("Wysyłanie żądania BOOTSTRAP do UDS", map[string]any{
		"target_service": cfg.TargetService,
		"services":       requiredServices,
		"socket_path":    cfg.SocketPath,
	})

	rawResp, err := ExecCommand(ctx, cfg.SocketPath, cfg.Timeout, cmdPayload)
	if err != nil {
		log.WarnObj("Błąd podczas ExecCommand w BootstrapApp", map[string]any{"err": err.Error()})
		return nil, func() {}, fmt.Errorf("bootstrap nie powiódł się: %w", err)
	}

	// Podgląd odebranego surowego bufora
	log.InfoMap("Odebrano odpowiedź z UDS", map[string]any{
		"raw_bytes_len": len(rawResp),
		"raw_string":    string(rawResp), // Może pokazać tekstowy komunikat błędu jeśli UDS zwrócił tekst zamiast msgpacka
	})

	var response FullBootstrapResponse
	if err := msgpack.Unmarshal(rawResp, &response); err != nil {
		log.WarnObj("Błąd deserializacji MessagePack w BootstrapApp", map[string]any{
			"err":       err.Error(),
			"raw_bytes": string(rawResp),
		})
		clear(rawResp)
		return nil, func() {}, fmt.Errorf("błąd deserializacji binarnej response: %w", err)
	}

	log.InfoMap("Zdeserializowano odpowiedź Bootstrap", map[string]any{
		"has_postgres": response.Postgres != nil,
		"has_redis":    response.Redis != nil,
		"has_minio":    response.Minio != nil,
		"has_rabbitmq": response.RabbitMQ != nil,
	})

	cleanup := func() {
		clear(rawResp)
		if response.Postgres != nil {
			clear(response.Postgres.Password)
		}
		if response.Redis != nil {
			clear(response.Redis.Password)
		}
		if response.Minio != nil {
			clear(response.Minio.SecretKey)
		}
		if response.RabbitMQ != nil {
			clear(response.RabbitMQ.Password)
		}
	}

	return &response, cleanup, nil
}

// RefreshPostgres pobiera odświeżone dane wyłącznie dla bazy danych Postgres.
func RefreshPostgres(ctx context.Context, cfg Config) (*PostgresCredentials, func(), error) {
	log := GetLogger()
	cmdPayload := fmt.Appendf(nil, "REFRESH %s postgres", cfg.TargetService)

	rawResp, err := ExecCommand(ctx, cfg.SocketPath, cfg.Timeout, cmdPayload)
	if err != nil {
		return nil, func() {}, fmt.Errorf("rotacja poświadczeń DB nie powiodła się: %w", err)
	}

	var creds PostgresCredentials
	if err := msgpack.Unmarshal(rawResp, &creds); err != nil {
		log.WarnObj("Błąd deserializacji Postgres credentials", map[string]any{
			"err":       err.Error(),
			"raw_bytes": string(rawResp),
		})
		clear(rawResp)
		return nil, func() {}, fmt.Errorf("błąd deserializacji postgres creds: %w", err)
	}

	cleanup := func() {
		clear(rawResp)
		clear(creds.Password)
	}

	return &creds, cleanup, nil
}

// ExecCommand realizuje binarny protokół UDS IPC w standardzie Length-Delimited.
func ExecCommand(ctx context.Context, socketPath string, timeout time.Duration, commandPayload []byte) ([]byte, error) {
	log := GetLogger()

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

	log.InfoMap("UDS ExecCommand - Wysyłanie ramki", map[string]any{
		"payload_raw": string(commandPayload),
		"payload_len": reqLen,
	})

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
		clear(respBuf)
		return nil, fmt.Errorf("błąd odczytu bajtów payloadu z UDS: %w", err)
	}

	return respBuf, nil
}

// FetchAgentSecret pobiera pojedynczy sekret bezpośrednio z cache sidecara,
// omijając komendę BOOTSTRAP i jej sztywne mapowania nazw usług.
func FetchAgentSecret(ctx context.Context, cfg Config, cacheKey string) (*PostgresCredentials, func(), error) {
	log := GetLogger()

	// Wysyłamy sam klucz bez prefixu komendy (np. "postgres_auth_auth_database")
	cmdPayload := []byte(cacheKey)

	log.InfoMap("Pobieranie sekretu z UDS (tryb bezpośredni)", map[string]any{
		"cache_key":   cacheKey,
		"socket_path": cfg.SocketPath,
	})

	rawResp, err := ExecCommand(ctx, cfg.SocketPath, cfg.Timeout, cmdPayload)
	if err != nil {
		return nil, func() {}, fmt.Errorf("błąd pobierania sekretu %q: %w", cacheKey, err)
	}

	var creds PostgresCredentials
	if err := msgpack.Unmarshal(rawResp, &creds); err != nil {
		log.WarnObj("Błąd deserializacji MessagePack", map[string]any{
			"err":       err.Error(),
			"cache_key": cacheKey,
		})
		clear(rawResp)
		return nil, func() {}, fmt.Errorf("deserializacja poświadczeń nie powiodła się: %w", err)
	}

	cleanup := func() {
		clear(rawResp)
		clear(creds.Password)
	}

	return &creds, cleanup, nil
}
