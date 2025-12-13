
---

### **`internal/` – logika aplikacji**

* **`handler/user_handler.go`** – warstwa HTTP, obsługa żądań przychodzących od klientów (endpointy, zwracanie JSON).

* **`model/user.go`** – struktury danych (np. `User`), mapowane na tabele w bazie danych.

* **`repository/mysql/mysql.go`** – dostęp do bazy danych, implementacja CRUD. Komunikacja z MySQL przez GORM.

* **`repository/repository.go`** – ogólne interfejsy repozytoriów (np. `UserRepository`), które definiują kontrakty, a konkretna implementacja jest w `mysql`.

* **`router/routes.go`** – definiowanie ścieżek HTTP i powiązanie ich z handlerami (`/users`, `/login` itp.).

* **`service/service.go`** – logika biznesowa (np. rejestracja, weryfikacja hasła, logika 2FA).

* **`service/password.go`** – helpery do hashowania i weryfikacji haseł.

* **`shared/db/db.go`** – konfiguracja i inicjalizacja połączenia z bazą danych (GORM).

* **`shared/logger/logger.go`** – konfiguracja loggera (zap), logowanie błędów, informacji, debug.

---
