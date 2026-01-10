### 1. Standardowe mechanizmy Go (Wbudowane)

Używaj ich głównie do tworzenia błędów lub szybkiego debugowania "na brudno".

| Narzędzie         | Przykład użycia                    | Kiedy stosować?                                                                                                       |
| ----------------- | ---------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| **`fmt.Errorf`**  | `fmt.Errorf("not found: %w", err)` | **Zawsze**, gdy chcesz zwrócić błąd z funkcji dalej "do góry". Operator `%w` pozwala na "owinięcie" błędu (wrapping). |
| **`panic`**       | `panic("critical database error")` | Tylko w sytuacjach **beznadziejnych**, np. gdy aplikacja nie może wystartować (brak `.env`).                          |
| **`fmt.Println`** | `fmt.Println("Tu dotarłem")`       | **Tylko tymczasowo** podczas pisania kodu. Usuń przed commitem!                                                       |

---

### 2. Twój Profesjonalny Logger (`pkg/shared`)

Twój logger to wrapper nad `uber-go/zap`. Jest inteligentny: sam maskuje hasła i zamienia błędy na pola JSON.

#### A. Metody Podstawowe (Używaj najczęściej)

Dzięki Twojej funkcji `parseArgs`, możesz tu wrzucać błędy, mapy i obiekty naraz.

- **`log.Debug(msg, args...)`**: Logi deweloperskie, niewidoczne na produkcji.
- **`log.Info(msg, args...)`**: Standardowe informacje o działaniu systemu.
- **`log.Warn(msg, args...)`**: Coś jest nie tak, ale system działa dalej.
- **`log.Error(msg, args...)`**: Błąd operacji (np. nieudane logowanie).
- **`log.Fatal(msg, args...)`**: Krytyczny błąd – wypisuje log i **zabija aplikację** (`os.Exit(1)`).

**Przykład "Professional":**

```go
// Automatycznie obsłuży błąd i mapę danych
log.Error("Failed to create user", err, map[string]any{"email": "test@wp.pl"})

```

#### B. Metody Specjalistyczne (Dla wygody i czystości)

| Metoda                        | Opis                                                                                    |
| ----------------------------- | --------------------------------------------------------------------------------------- |
| **`ErrorObj(msg, obj)`**      | Przyjmuje dowolną strukturę (np. model GORM). Sam wyciągnie pola przez refleksję.       |
| **`InfoMap(msg, map)`**       | Idealne dla logowania parametrów requestu. Automatycznie **maskuje hasła**.             |
| **`DebugResponse(msg, res)`** | Specjalna metoda, która drukuje wynik w konsoli w **kolorowych ramkach**. Super do API. |
| **`DebugEmpty(msg, key)`**    | Loguje informację, że dany klucz jest pusty (używa symbolu `∅`).                        |

---

### 3. Maskowanie i Bezpieczeństwo

Twój logger posiada funkcję `isSensitive`. Jeśli logujesz mapę lub obiekt, który zawiera klucze takie jak:
`password`, `token`, `secret`, `authorization`, `apikey`

Zostaną one automatycznie zamienione na `********`.

---

### 4. Kiedy czego użyć? (Scenariusze)

#### Scenariusz A: Brak pliku `.env` przy starcie

Używamy `log.Fatal`, bo bez tego serwis nie ma sensu.

```go
if err := viper.ReadInConfig(); err != nil {
    log.Fatal("Shutting down: Config file not found", err)
}

```

#### Scenariusz B: Błąd zapytania do bazy danych

Używamy `log.ErrorObj`, żeby widzieć co to za błąd i dla jakiego obiektu wystąpił.

```go
if err := db.Create(&user).Error; err != nil {
    log.ErrorObj("Database insert failed", user)
    return err
}

```

#### Scenariusz C: Debugowanie odpowiedzi z innego mikroserwisu

Używamy `DebugResponse` dla maksymalnej czytelności w konsoli.

```go
log.DebugResponse("Auth service response", responseBody)

```

---

### Podsumowanie dla Programisty:

1. W funkcjach **zwracaj** błędy przez `fmt.Errorf`.
2. W `main.go` lub w handlerach **loguj** błędy przez `log.Error(msg, err)`.
3. Nigdy nie loguj surowych haseł (Twój logger Cię przed tym chroni, ale miej to na uwadze).
4. Pamiętaj: `log.Fatal` to koniec programu – używaj tylko przy starcie aplikacji.
