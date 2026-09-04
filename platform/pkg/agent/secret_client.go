package agent

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/zerodayz7/platform/pkg/shared"
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

// BootstrapApp pobiera komplet sekretów przy użyciu protokołu MessagePack przez UDS.
func BootstrapApp(ctx context.Context, cfg Config, requiredServices []string) (*FullBootstrapResponse, func(), error) {
	log := shared.GetLogger()

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

	var response FullBootstrapResponse
	if err := msgpack.Unmarshal(rawResp, &response); err != nil {
		log.WarnObj("Błąd deserializacji MessagePack w BootstrapApp", map[string]any{
			"err":       err.Error(),
			"raw_bytes": string(rawResp),
		})
		clear(rawResp)
		return nil, func() {}, fmt.Errorf("błąd deserializacji binarnej response: %w", err)
	}

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

// region FetchAgentSecret
// FetchAgentSecret pobiera pojedynczy odświeżony sekret bezpośrednio z pamięci Agent-Sidecara.
func FetchAgentSecret[T any](ctx context.Context, socketPath string, timeout time.Duration, cacheKey string) (*T, func(), error) {
	log := shared.GetLogger()
	cmdPayload := []byte(cacheKey)

	log.InfoMap("🔍 Requesting secret from Agent IPC", map[string]any{
		"cache_key":   cacheKey,
		"socket_path": socketPath,
		"timeout":     timeout.String(),
	})

	start := time.Now()
	rawResp, err := ExecCommand(ctx, socketPath, timeout, cmdPayload)
	duration := time.Since(start)

	if err != nil {
		log.WarnObj("❌ ExecCommand failed in FetchAgentSecret", map[string]any{
			"cache_key":   cacheKey,
			"socket_path": socketPath,
			"duration":    duration.String(),
			"err":         err.Error(),
		})
		return nil, func() {}, fmt.Errorf("błąd pobierania sekretu [%s]: %w", cacheKey, err)
	}

	log.InfoMap("📦 Received raw payload from Agent UDS", map[string]any{
		"cache_key": cacheKey,
		"bytes_len": len(rawResp),
		"duration":  duration.String(),
	})

	var creds T
	if err := msgpack.Unmarshal(rawResp, &creds); err != nil {
		log.WarnObj("❌ MessagePack unmarshal error", map[string]any{
			"err":       err.Error(),
			"cache_key": cacheKey,
			"raw_hex":   fmt.Sprintf("%x", rawResp),
		})
		clear(rawResp)
		return nil, func() {}, fmt.Errorf("deserializacja poświadczeń [%s] nie powiodła się: %w", cacheKey, err)
	}

	cleanup := func() {
		clear(rawResp)
	}

	return &creds, cleanup, nil
}

// region ExecCommand
// ExecCommand realizuje binarny protokół UDS IPC w standardzie Length-Delimited.
func ExecCommand(ctx context.Context, socketPath string, timeout time.Duration, commandPayload []byte) ([]byte, error) {
	log := shared.GetLogger()

	// Sprawdzamy czy plik gniazda w ogóle istnieje na dysku kontenera przed próbą dial
	if info, err := os.Stat(socketPath); err != nil {
		log.WarnObj("❌ Socket file check failed on filesystem", map[string]any{
			"socket_path": socketPath,
			"err":         err.Error(),
		})
	} else {
		log.InfoMap("🔌 Socket file present on filesystem", map[string]any{
			"socket_path": socketPath,
			"mode":        info.Mode().String(),
		})
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("błąd połączenia z UDS [%s]: %w", socketPath, err)
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
		clear(respBuf)
		return nil, fmt.Errorf("błąd odczytu bajtów payloadu z UDS: %w", err)
	}

	return respBuf, nil
}
