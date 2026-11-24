package auth

import "testing"

func TestSignAndVerify(t *testing.T) {
	signature := sign("user1")

	if !verify("user1", signature) {
		t.Fatal("правильная подпись не прошла проверку")
	}

	if verify("user2", signature) {
		t.Fatal("подпись принята для изменённого ID пользователя")
	}
}

func TestNewCookieAndParse(t *testing.T) {
	cookie := NewCookie()

	if cookie.Name != "user_id" {
		t.Fatalf("получили имя cookie %q, ожидали user_id", cookie.Name)
	}

	userID, err := ParseCookie(cookie.Value)
	if err != nil {
		t.Fatalf("не удалось проверить созданную cookie: %v", err)
	}

	if userID == "" {
		t.Fatal("из cookie получен пустой ID пользователя")
	}
}

func TestParseCookieInvalidSignature(t *testing.T) {
	cookie := NewCookie()

	_, err := ParseCookie(cookie.Value + "x")
	if err != ErrInvalidSignature {
		t.Fatalf(
			"получили ошибку %v, ожидали %v",
			err,
			ErrInvalidSignature,
		)
	}
}
