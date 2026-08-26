package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/zerodayz7/platform/pkg/crypto"
	"github.com/zerodayz7/platform/pkg/shared"
	"github.com/zerodayz7/platform/services/citizen-docs/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	testUserID1 = uuid.MustParse("707a8869-6867-4601-9337-e23fcb51b0ad")
	testUserID2 = uuid.MustParse("a2f6b8c9-1122-4a55-8822-b98765432101")
)

func mockSignature(payload string) []byte {
	sig, _ := base64.StdEncoding.DecodeString(base64.StdEncoding.EncodeToString([]byte("MOCK_SIGNATURE_" + payload)))
	return sig
}

func mockImageData(name string) []byte {
	return []byte("MOCK_IMAGE_BYTES_FOR_" + name)
}

func mockDEK(name string) []byte {
	return []byte("MOCK_DEK_FOR_" + name)
}

func SeedData(db *gorm.DB) error {
	log := shared.GetLogger()

	var count int64
	if err := db.Model(&model.CitizenProfile{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check citizen profiles count: %w", err)
	}

	if count > 0 {
		return nil
	}

	log.Info("[SEED] Rozpoczynam rozbudowane zasiewanie bazy citizen-docs...")

	now := time.Now()
	seedSecret := []byte(AppConfig.Security.DocsPeselSalt)

	// ==========================================
	// PROFIL 1: Główny użytkownik testowy (Jan Kowalski)
	// ==========================================
	profileID1, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate profile1 uuidv7: %w", err)
	}

	rawCitizenData1 := model.CitizenData{
		FirstName:   "Jan",
		LastName:    "Kowalski",
		PESEL:       "90010112345",
		DateOfBirth: "1990-01-01",
		Citizenship: "PL",
		Attributes:  datatypes.JSON([]byte(`{"gender":"M","blood_type":"A+","eye_color":"blue","height":182}`)),
	}
	bytesCitizenData1, err := json.Marshal(rawCitizenData1)
	if err != nil {
		return fmt.Errorf("failed to marshal seed citizen data 1: %w", err)
	}

	// 1.1 Dowód osobisty (Aktywny)
	metaIDCard := model.DocumentMeta{
		DocumentNumber:   "ABA123456",
		Title:            "Dowód Osobisty",
		Issuer:           "PREZYDENT MIASTA KATOWICE",
		Category:         "identity",
		AllowedScopes:    []string{"full_identity", "age_verification"},
		CustomAttributes: datatypes.JSON([]byte(`{"organ_wydajacy_kod":"2467","can_number":"123456"}`)),
	}
	bytesMetaID, _ := json.Marshal(metaIDCard)
	issuedAtID := now.AddDate(-2, 0, 0)
	expiresAtID := now.AddDate(8, 0, 0)

	docID1, _ := uuid.NewV7()
	doc1 := model.UserDocument{
		ID:                docID1,
		ProfileID:         profileID1,
		TypeCode:          "ID_CARD",
		Status:            model.DocumentStatusActive,
		EncryptedMeta:     bytesMetaID,
		EncryptedMetaDEK:  mockDEK("id_card_meta"),
		EncryptedFront:    mockImageData("id_card_front"),
		EncryptedFrontDEK: mockDEK("id_card_front"),
		EncryptedBack:     mockImageData("id_card_back"),
		EncryptedBackDEK:  mockDEK("id_card_back"),
		IssuerSignature:   mockSignature("ID_CARD_JAN_KOWALSKI"),
		SigningKeyID:      "GOV-PL-KMS-2024-KEY-01",
		RevocationSerial:  "REV-ID-2024-009812",
		Version:           1,
		IssuedAt:          &issuedAtID,
		ExpiresAt:         &expiresAtID,
	}

	// 1.2 Prawo Jazdy (Aktywne)
	metaDriverLicense := model.DocumentMeta{
		DocumentNumber:   "12345/22/2401",
		Title:            "Prawo Jazdy",
		Issuer:           "STAROSTA BĘDZIŃSKI",
		Category:         "qualification",
		AllowedScopes:    []string{"driving_privileges"},
		CustomAttributes: datatypes.JSON([]byte(`{"categories":["AM","B","B1"],"restrictions":["01.06"]}`)),
	}
	bytesMetaDL, _ := json.Marshal(metaDriverLicense)
	issuedAtDL := now.AddDate(-1, -3, 0)
	expiresAtDL := now.AddDate(14, 0, 0)

	docID2, _ := uuid.NewV7()
	doc2 := model.UserDocument{
		ID:                docID2,
		ProfileID:         profileID1,
		TypeCode:          "DRIVER_LICENSE",
		Status:            model.DocumentStatusActive,
		EncryptedMeta:     bytesMetaDL,
		EncryptedMetaDEK:  mockDEK("dl_meta"),
		EncryptedFront:    mockImageData("dl_front"),
		EncryptedFrontDEK: mockDEK("dl_front"),
		EncryptedBack:     mockImageData("dl_back"),
		EncryptedBackDEK:  mockDEK("dl_back"),
		IssuerSignature:   mockSignature("DL_JAN_KOWALSKI"),
		SigningKeyID:      "GOV-PL-KMS-2024-KEY-02",
		RevocationSerial:  "REV-DL-2024-004112",
		Version:           2,
		IssuedAt:          &issuedAtDL,
		ExpiresAt:         &expiresAtDL,
	}

	// 1.3 Karta Dużej Rodziny (Aktywna)
	metaLargeFamilyCard := model.DocumentMeta{
		DocumentNumber:   "KDR-900101-01",
		Title:            "Karta Dużej Rodziny",
		Issuer:           "MINISTER WŁAŚCIWY DO SPRAW RODZINY",
		Category:         "social",
		CustomAttributes: datatypes.JSON([]byte(`{"children_count":3,"role":"PARENT"}`)),
	}
	bytesMetaKDR, _ := json.Marshal(metaLargeFamilyCard)
	issuedAtKDR := now.AddDate(-3, 0, 0)

	docID3, _ := uuid.NewV7()
	doc3 := model.UserDocument{
		ID:                docID3,
		ProfileID:         profileID1,
		TypeCode:          "LARGE_FAMILY_CARD",
		Status:            model.DocumentStatusActive,
		EncryptedMeta:     bytesMetaKDR,
		EncryptedMetaDEK:  mockDEK("kdr_meta"),
		EncryptedFront:    mockImageData("kdr_front"),
		EncryptedFrontDEK: mockDEK("kdr_front"),
		IssuerSignature:   mockSignature("KDR_JAN_KOWALSKI"),
		SigningKeyID:      "GOV-PL-KMS-2023-KEY-05",
		RevocationSerial:  "REV-KDR-2023-991201",
		Version:           1,
		IssuedAt:          &issuedAtKDR,
		ExpiresAt:         nil,
	}

	profile1 := model.CitizenProfile{
		ID:            profileID1,
		UserID:        testUserID1,
		EncryptedData: bytesCitizenData1,
		EncryptedDEK:  mockDEK("profile_jan"),
		PeselHash:     crypto.ComputeHMAC256Hex([]byte("90010112345"), seedSecret),
		Documents:     []model.UserDocument{doc1, doc2, doc3},
	}

	if err := db.Create(&profile1).Error; err != nil {
		return fmt.Errorf("failed to seed citizen profile 1: %w", err)
	}

	// ==========================================
	// PROFIL 2: Użytkownik brzegowy (Anna Nowak)
	// ==========================================
	profileID2, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate profile2 uuidv7: %w", err)
	}

	rawCitizenData2 := model.CitizenData{
		FirstName:   "Anna",
		LastName:    "Nowak",
		PESEL:       "95050554321",
		DateOfBirth: "1995-05-05",
		Citizenship: "PL",
		Attributes:  datatypes.JSON([]byte(`{"gender":"F","blood_type":"0+"}`)),
	}
	bytesCitizenData2, err := json.Marshal(rawCitizenData2)
	if err != nil {
		return fmt.Errorf("failed to marshal seed citizen data 2: %w", err)
	}

	// 2.1 Legitymacja Studencka (Wygasła)
	metaStudent := model.DocumentMeta{
		DocumentNumber:   "ELS-2194812",
		Title:            "Elektroniczna Legitymacja Studencka",
		Issuer:           "POLITECHNIKA ŚLĄSKA",
		Category:         "education",
		CustomAttributes: datatypes.JSON([]byte(`{"faculty":"AEI","field_of_study":"Informatyka"}`)),
	}
	bytesMetaStudent, _ := json.Marshal(metaStudent)
	issuedAtStudent := now.AddDate(-4, 0, 0)
	expiresAtStudent := now.AddDate(-1, 0, 0)

	docID4, _ := uuid.NewV7()
	doc4 := model.UserDocument{
		ID:                docID4,
		ProfileID:         profileID2,
		TypeCode:          "STUDENT_ID",
		Status:            model.DocumentStatusExpired,
		EncryptedMeta:     bytesMetaStudent,
		EncryptedMetaDEK:  mockDEK("student_meta"),
		EncryptedFront:    mockImageData("student_id_front"),
		EncryptedFrontDEK: mockDEK("student_id_front"),
		EncryptedBack:     mockImageData("student_id_back"),
		EncryptedBackDEK:  mockDEK("student_id_back"),
		IssuerSignature:   mockSignature("STUDENT_ANNA_NOWAK"),
		SigningKeyID:      "POLSL-KMS-2022-KEY-01",
		RevocationSerial:  "REV-ELS-2022-000102",
		Version:           1,
		IssuedAt:          &issuedAtStudent,
		ExpiresAt:         &expiresAtStudent,
	}

	// 2.2 Unieważniony Dowód Osobisty
	metaRevokedID := model.DocumentMeta{
		DocumentNumber: "XYZ987654",
		Title:          "Dowód Osobisty",
		Issuer:         "PREZYDENT MIASTA OPOLE",
		Category:       "identity",
	}
	bytesMetaRevoked, _ := json.Marshal(metaRevokedID)
	issuedAtRev := now.AddDate(-5, 0, 0)
	expiresAtRev := now.AddDate(5, 0, 0)

	docID5, _ := uuid.NewV7()
	doc5 := model.UserDocument{
		ID:                docID5,
		ProfileID:         profileID2,
		TypeCode:          "ID_CARD",
		Status:            model.DocumentStatusRevoked,
		EncryptedMeta:     bytesMetaRevoked,
		EncryptedMetaDEK:  mockDEK("revoked_meta"),
		EncryptedFront:    mockImageData("revoked_id_front"),
		EncryptedFrontDEK: mockDEK("revoked_id_front"),
		EncryptedBack:     mockImageData("revoked_id_back"),
		EncryptedBackDEK:  mockDEK("revoked_id_back"),
		IssuerSignature:   mockSignature("REVOKED_ID_ANNA_NOWAK"),
		SigningKeyID:      "GOV-PL-KMS-2021-KEY-09",
		RevocationSerial:  "REV-ID-2023-999999",
		Version:           3,
		IssuedAt:          &issuedAtRev,
		ExpiresAt:         &expiresAtRev,
	}

	profile2 := model.CitizenProfile{
		ID:            profileID2,
		UserID:        testUserID2,
		EncryptedData: bytesCitizenData2,
		EncryptedDEK:  mockDEK("profile_anna"),
		PeselHash:     crypto.ComputeHMAC256Hex([]byte("95050554321"), seedSecret),
		Documents:     []model.UserDocument{doc4, doc5},
	}

	if err := db.Create(&profile2).Error; err != nil {
		return fmt.Errorf("failed to seed citizen profile 2: %w", err)
	}

	log.Info(fmt.Sprintf("[SEED] Zasiano bazę danymi testowymi! Utworzono 2 profile oraz %d dokumentów.", len(profile1.Documents)+len(profile2.Documents)))
	return nil
}
