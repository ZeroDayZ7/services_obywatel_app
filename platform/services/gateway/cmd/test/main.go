package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/zerodayz7/platform/pkg/kms"
	"github.com/zerodayz7/platform/services/gateway/config"
)

func main() {
	if err := config.LoadConfigGlobal(); err != nil {
		log.Fatalf("[BŁĄD] Nie udało się załadować konfiguracji Gateway: %v", err)
	}

	kmsCfg := config.AppConfig.ToKMSServiceConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	keyAlias := "master_key"

	fmt.Println("===> 1. Pobieranie DataKey (DEK) z vHSM/KMS...")
	dataKey, err := kms.GenerateDataKey(ctx, kmsCfg, keyAlias)
	if err != nil {
		log.Fatalf("[BŁĄD] Nie udało się wygenerować DataKey: %v", err)
	}

	defer kms.ZeroBytes(dataKey.Plaintext)

	fmt.Println("\n===> [SUKCES] Pobrano DataKey z KMS!")
	fmt.Printf(" - Master Key Version: %d\n", dataKey.MasterKeyVersion)

	if len(dataKey.Plaintext) >= 4 {
		fmt.Printf(" - Plaintext DEK (długość: %d bajtów): %x...\n", len(dataKey.Plaintext), dataKey.Plaintext[:4])
	} else {
		fmt.Printf(" - Plaintext DEK (długość: %d bajtów): %x\n", len(dataKey.Plaintext), dataKey.Plaintext)
	}

	if len(dataKey.Ciphertext) >= 8 {
		fmt.Printf(" - Ciphertext DEK (wrapped, długość: %d bajtów): %x...\n", len(dataKey.Ciphertext), dataKey.Ciphertext[:8])
	} else {
		fmt.Printf(" - Ciphertext DEK (wrapped, długość: %d bajtów): %x\n", len(dataKey.Ciphertext), dataKey.Ciphertext)
	}

	secretMessage := []byte("Tajny tekst do zaszyfrowania za pomocą DEK z vHSM!")
	fmt.Printf("\n===> 2. Szyfrowanie wiadomości: \"%s\"\n", string(secretMessage))

	ciphertextPayload, nonce, err := encryptAESGCM(dataKey.Plaintext, secretMessage)
	if err != nil {
		log.Fatalf("[BŁĄD] Szyfrowanie lokalne AES-GCM nie powiodło się: %v", err)
	}
	fmt.Printf(" - Zaszyfrowana wiadomość (Hex): %x\n", ciphertextPayload)
	fmt.Printf(" - Nonce/IV (Hex): %x\n", nonce)

	fmt.Println("\n===> 3. Odszyfrowywanie wiadomości...")
	decryptedMessage, err := decryptAESGCM(dataKey.Plaintext, nonce, ciphertextPayload)
	if err != nil {
		log.Fatalf("[BŁĄD] Odszyfrowanie lokalne AES-GCM nie powiodło się: %v", err)
	}

	fmt.Printf(" - Odszyfrowana treść: \"%s\"\n", string(decryptedMessage))
	fmt.Println("\n Status: Test przebiegł pomyślnie!")
}

func encryptAESGCM(key []byte, plaintext []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

func decryptAESGCM(key []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
