package config

import (
	"crypto/rand"
	"fmt"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/messaging-service/internal/model"
	"gorm.io/gorm"
)

var (
	testUserID1 = uuid.MustParse("707a8869-6867-4601-9337-e23fcb51b0ad") // Jan Kowalski
	testUserID2 = uuid.MustParse("a2f6b8c9-1122-4a55-8822-b98765432101") // Anna Nowak
	testUserID3 = uuid.MustParse("c3d4e5f6-3344-5b66-9933-a12345678902") // Piotr Wiśniewski
)

func mockBytes(size int, prefix string) []byte {
	buf := make([]byte, size)
	_, _ = rand.Read(buf)
	copy(buf, []byte(prefix))
	return buf
}

func SeedData(db *gorm.DB) error {
	log := shared.GetLogger()

	var count int64
	if err := db.Model(&model.UserDeviceIdentity{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check messaging seed state: %w", err)
	}

	if count > 0 {
		return nil
	}

	log.Info("[SEED] Rozpoczynam zasiewanie bazy danych messaging-service (E2EE / Chats / Contacts)...")

	// ==========================================
	// 1. KRYPTOGRAFIA (E2EE Device Identity & PreKeys)
	// ==========================================

	// Dev1 - Jan Kowalski
	devIdentity1 := model.UserDeviceIdentity{
		ID:                  uuid.Must(uuid.NewV7()),
		UserID:              testUserID1,
		DeviceID:            "DEV-IOS-JAN-01",
		PublicKey:           mockBytes(32, "PUBKEY_JAN_IDENTITY"),
		SignedPreKey:        mockBytes(32, "SIGNED_PREKEY_JAN"),
		SignedPreKeySig:     mockBytes(64, "SIG_JAN"),
		SignedPreKeyID:      1,
		OneTimePreKeysCount: 5,
	}

	// Dev2 - Anna Nowak
	devIdentity2 := model.UserDeviceIdentity{
		ID:                  uuid.Must(uuid.NewV7()),
		UserID:              testUserID2,
		DeviceID:            "DEV-ANDROID-ANNA-01",
		PublicKey:           mockBytes(32, "PUBKEY_ANNA_IDENTITY"),
		SignedPreKey:        mockBytes(32, "SIGNED_PREKEY_ANNA"),
		SignedPreKeySig:     mockBytes(64, "SIG_ANNA"),
		SignedPreKeyID:      1,
		OneTimePreKeysCount: 5,
	}

	if err := db.Create(&devIdentity1).Error; err != nil {
		return fmt.Errorf("failed to seed device identity 1: %w", err)
	}
	if err := db.Create(&devIdentity2).Error; err != nil {
		return fmt.Errorf("failed to seed device identity 2: %w", err)
	}

	// PreKeys dla Jana Kowalskiego
	for i := uint32(1); i <= 5; i++ {
		preKey := model.UserPreKey{
			ID:        uuid.Must(uuid.NewV7()),
			DeviceID:  devIdentity1.ID,
			KeyID:     i,
			PublicKey: mockBytes(32, fmt.Sprintf("OTK_JAN_%d", i)),
		}
		if err := db.Create(&preKey).Error; err != nil {
			return fmt.Errorf("failed to seed prekey for user 1: %w", err)
		}
	}

	// ==========================================
	// 2. DOMENA KONTAKTY (Contacts)
	// ==========================================

	// Jan Kowalski posiada w kontaktach Annę i Piotra
	contact1 := model.Contact{
		ID:             uuid.Must(uuid.NewV7()),
		OwnerID:        testUserID1,
		ContactID:      testUserID2,
		Status:         model.ContactStatusAccepted,
		EncryptedAlias: []byte("ZaszyfrowanyAlias_Anna"),
		Version:        1,
	}

	contact2 := model.Contact{
		ID:             uuid.Must(uuid.NewV7()),
		OwnerID:        testUserID1,
		ContactID:      testUserID3,
		Status:         model.ContactStatusPending,
		EncryptedAlias: []byte("ZaszyfrowanyAlias_Piotr"),
		Version:        1,
	}

	// Anna ma w kontaktach Jana
	contact3 := model.Contact{
		ID:             uuid.Must(uuid.NewV7()),
		OwnerID:        testUserID2,
		ContactID:      testUserID1,
		Status:         model.ContactStatusAccepted,
		EncryptedAlias: []byte("ZaszyfrowanyAlias_Jan"),
		Version:        1,
	}

	if err := db.Create(&[]model.Contact{contact1, contact2, contact3}).Error; err != nil {
		return fmt.Errorf("failed to seed contacts: %w", err)
	}

	// ==========================================
	// 3. CHATY I KONWERSACJE (Direct & Group)
	// ==========================================

	// 3.1 Konwersacja Direct (Jan <-> Anna)
	convDirectID := uuid.Must(uuid.NewV7())
	convDirect := model.Conversation{
		ID:           convDirectID,
		Type:         model.ConversationTypeDirect,
		LastSequence: 3,
	}
	if err := db.Create(&convDirect).Error; err != nil {
		return fmt.Errorf("failed to seed direct conversation: %w", err)
	}

	memberDirect1 := model.ConversationMember{
		ID:               uuid.Must(uuid.NewV7()),
		ConversationID:   convDirectID,
		UserID:           testUserID1,
		Role:             "member",
		LastReadSequence: 3,
	}
	memberDirect2 := model.ConversationMember{
		ID:               uuid.Must(uuid.NewV7()),
		ConversationID:   convDirectID,
		UserID:           testUserID2,
		Role:             "member",
		LastReadSequence: 2, // Anna nie przeczytała jeszcze 3 wiadomości
	}
	if err := db.Create(&[]model.ConversationMember{memberDirect1, memberDirect2}).Error; err != nil {
		return fmt.Errorf("failed to seed direct conversation members: %w", err)
	}

	// Wiadomości w konwersacji direct
	msg1 := model.Message{
		ID:               uuid.Must(uuid.NewV7()),
		ConversationID:   convDirectID,
		SenderID:         testUserID1,
		SenderDeviceID:   devIdentity1.DeviceID,
		Type:             model.MessageTypeText,
		Sequence:         1,
		EncryptedPayload: mockBytes(128, "E2EE_PAYLOAD_CZESC_ANNA"),
		Version:          1,
	}
	msg2 := model.Message{
		ID:               uuid.Must(uuid.NewV7()),
		ConversationID:   convDirectID,
		SenderID:         testUserID2,
		SenderDeviceID:   devIdentity2.DeviceID,
		Type:             model.MessageTypeText,
		Sequence:         2,
		EncryptedPayload: mockBytes(128, "E2EE_PAYLOAD_HEJ_JAN"),
		Version:          2,
	}
	msg3 := model.Message{
		ID:               uuid.Must(uuid.NewV7()),
		ConversationID:   convDirectID,
		SenderID:         testUserID1,
		SenderDeviceID:   devIdentity1.DeviceID,
		Type:             model.MessageTypeMedia,
		Sequence:         3,
		EncryptedPayload: mockBytes(256, "E2EE_PAYLOAD_IMAGE_ATTACHMENT"),
		MediaHeader:      mockBytes(64, "MEDIA_KEY_HEADER"),
		Version:          3,
	}
	if err := db.Create(&[]model.Message{msg1, msg2, msg3}).Error; err != nil {
		return fmt.Errorf("failed to seed direct messages: %w", err)
	}

	// 3.2 Konwersacja Grupowa (Jan + Anna + Piotr)
	convGroupID := uuid.Must(uuid.NewV7())
	convGroup := model.Conversation{
		ID:           convGroupID,
		Type:         model.ConversationTypeGroup,
		Title:        "Projekt Obywatel Plus",
		LastSequence: 1,
	}
	if err := db.Create(&convGroup).Error; err != nil {
		return fmt.Errorf("failed to seed group conversation: %w", err)
	}

	groupMembers := []model.ConversationMember{
		{
			ID:               uuid.Must(uuid.NewV7()),
			ConversationID:   convGroupID,
			UserID:           testUserID1,
			Role:             "admin",
			LastReadSequence: 1,
		},
		{
			ID:               uuid.Must(uuid.NewV7()),
			ConversationID:   convGroupID,
			UserID:           testUserID2,
			Role:             "member",
			LastReadSequence: 1,
		},
		{
			ID:               uuid.Must(uuid.NewV7()),
			ConversationID:   convGroupID,
			UserID:           testUserID3,
			Role:             "member",
			LastReadSequence: 0,
		},
	}
	if err := db.Create(&groupMembers).Error; err != nil {
		return fmt.Errorf("failed to seed group members: %w", err)
	}

	groupMsg1 := model.Message{
		ID:               uuid.Must(uuid.NewV7()),
		ConversationID:   convGroupID,
		SenderID:         testUserID1,
		SenderDeviceID:   devIdentity1.DeviceID,
		Type:             model.MessageTypeSystem,
		Sequence:         1,
		EncryptedPayload: mockBytes(64, "SYSTEM_GROUP_CREATED"),
		Version:          4,
	}
	if err := db.Create(&groupMsg1).Error; err != nil {
		return fmt.Errorf("failed to seed group message: %w", err)
	}

	log.Info("[SEED] Pomyślnie zasiano bazę danych messaging-service! Utworzono 2 tożsamości urządzeń, 3 kontakty, 2 konwersacje oraz 4 wiadomości z wersjonowaniem Delta Sync.")
	return nil
}
