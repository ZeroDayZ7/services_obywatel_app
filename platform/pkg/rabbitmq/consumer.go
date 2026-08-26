package rabbitmq

import (
	"context"
	"crypto/hmac"
	"fmt"

	"github.com/zerodayz7/platform/pkg/crypto"
	"github.com/zerodayz7/platform/pkg/httpserver"
	"github.com/zerodayz7/platform/pkg/shared"
)

// #region SubscribeWithAuth
func (p *RabbitMQPublisher) SubscribeWithAuth(
	ctx context.Context,
	queueName string,
	routingKey string,
	keyStore *httpserver.KeyStore,
	handler HandlerFunc,
) error {
	log := shared.GetLogger()

	msgs, err := p.Consume(queueName, routingKey)
	if err != nil {
		return fmt.Errorf("failed to start consuming queue %s: %w", queueName, err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				return nil
			}

			// Weryfikacja tożsamości i podpisu
			if keyStore != nil {
				senderID, _ := d.Headers["X-Sender-ID"].(string)
				signature, _ := d.Headers["X-Signature"].(string)

				expectedKey, _, ok := keyStore.GetKey(senderID)
				if !ok || !verifyHMAC(d.Body, expectedKey, signature) {
					log.Error("[RABBITMQ] Odrzucono wiadomość: niepoprawny podpis HMAC", "sender", senderID, "queue", queueName)
					_ = d.Nack(false, false)
					continue
				}
			}

			if err := handler(ctx, d.Headers, d.Body); err != nil {
				log.Error("[RABBITMQ] Błąd przetwarzania wiadomości", "queue", queueName, "error", err)
				_ = d.Nack(false, false)
				continue
			}

			_ = d.Ack(false)
		}
	}
}

// #region verifyHMAC
func verifyHMAC(payload []byte, key []byte, signature string) bool {
	expectedSig := crypto.ComputeHMAC256Hex(payload, key)
	return hmac.Equal([]byte(expectedSig), []byte(signature))
}
