package sshd

import "testing"

func TestLoadHostKey_EmptyValueIsOptional(t *testing.T) {
	option, err := loadHostKey("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if option != nil {
		t.Fatalf("expected no host key option when PORTR_SSH_HOST_KEY is unset")
	}
}

func TestLoadHostKey_GeneratedKeyIsAccepted(t *testing.T) {
	key, err := GenerateHostKey()
	if err != nil {
		t.Fatalf("failed to generate host key: %v", err)
	}

	option, err := loadHostKey(key)
	if err != nil {
		t.Fatalf("expected generated key to be accepted, got %v", err)
	}
	if option == nil {
		t.Fatalf("expected host key option for generated key")
	}
}
