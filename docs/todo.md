1. **Wybór stosu dla panelu rejestracji: Angular vs Flutter Web vs Next.js**
* *Angular:* Świetny do ciężkich, rozbudowanych paneli administracyjnych z architekturą modułową.
* *Flutter Web:* Daje 100% spójności kodu i UI z aplikacją mobilną, ale bywa cięższy przy pierwszym ładowaniu na desktopie.
* *Kwestia do rozważenia:* Czy robimy osobny panel www dla urzędników/administratorów, czy jeden uniwersalny interfejs użytkownika na Flutterze dla webu i mobile.


2. **Integracja z backendem w Go i modułem KMS (`kms-service`)**
* Jak panel/aplikacja komunikuje się z KMS do bezpiecznego podpisywania głosów, generowania kluczy i rewrappingu tokenów.
* Oddzielenie odpowiedzialności: backend Go jako API Gateway oraz orkiestrator biznesowy, a KMS jako odizolowana usługa do operacji kryptograficznych.


3. **Tożsamość i Zero-Trust Onboarding (Auth & eID)**
* Jak rejestrujemy i weryfikujemy obywatela/użytkownika (np. klucze prywatne w chmurze vs lokalny secure storage na telefonie).
* Weryfikacja tożsamości – czy integracja z węzłem krajowym/QR kodami, czy wewnętrzny mechanizm weryfikacji.


4. **Mechanizm Płynnego Delegowania Głosów (Liquid Delegation Graph)**
* Model danych w Go do reprezentowania grafu delegacji (kto komu przekazuje głos w jakiej kategorii, np. ekologia, finanse, infrastruktura).
* Wykrywanie i zapobieganie pętlom w grafie delegacji (A deleguje na B, B na C, C na A).


5. **Kryptograficzna Anonimowość i Weryfikowalność Głosowania**
* Wybór podejścia: dowody z wiedzą zerową (ZKP - Zero-Knowledge Proofs), ślepe podpisy (Blind Signatures) czy homomorficzne szyfrowanie.
* Zapewnienie dwóch kluczowych cech: użytkownik wie, że jego głos został zaliczony, ale nikt inny nie może sprawdzić, jak dokładnie zagłosował.


6. **Architektura Aplikacji Mobilnej we Flutterze**
* Podział na pakiety i stan aplikacji (np. BLoC/Cubit), wydzielenie warstwy kryptograficznej do natywnych modułów Rust/Go (mobile bindings) lub czytelnej obsługi kluczy na telefonie.
* Prywatny rejestr delegatów i przeglądanie uchwał/głosowań.


7. **Praca w trybie Offline i Synchronizacja (P2P / Local-First)**
* Wykorzystanie lokalnych baz danych na urządzeniu (np. Hive/Isar we Flutterze) do przeglądania kart głosowania offline.
* Bezpieczne kolejkowanie podpisanych głosów/delegacji i ich wysyłka po powrocie do sieci.


8. **Audytowalność i Niezaprzeczalność Rejestru Głosów**
* Struktura przechowywania kart głosowania (drzewa Merkle'a, lokalny append-only log czy rozproszona baza).
* Udostępnienie otwartego API w Go dla zewnętrznych audytorów, pozwalającego przeliczyć głosy i zweryfikować poprawność wyników bez ujawniania danych osobowych.


9. **Model Uprawnień i System Zapobiegania Sybil Attack**
* Ochrona przed tworzeniem fikcyjnych kont – powiązanie konta z unikalną tożsamością fizyczną.
* Limity zapytań (rate limiting), ochrona na poziomie sieciowym (Cilium/Wazuh) i monitoring nieprawidłowych wzorców delegowania.


10. **Strategia Wdrożeniowa i Cykl Życia Projektu**
* Podział na kroki: przygotowanie PoC (Proof of Concept) rejestracji i głosowania, testy obciążeniowe API w Go, a następnie integracja z panelem frontendowym i aplikacją mobilną.
* Konfiguracja środowisk deweloperskich (K3s/Docker) i bezpiecznego pipeline'u CI/CD.