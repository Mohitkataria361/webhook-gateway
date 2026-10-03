package tests

import (
	"testing"

	"webhook-gateway/internal/signer"
)

func TestSign_ProducesExpectedPrefix(t *testing.T) {
	sig := signer.Sign("mysecret", []byte(`{"event":"test"}`))
	if len(sig) == 0 {
		t.Fatal("expected non-empty signature")
	}
	if sig[:7] != "sha256=" {
		t.Errorf("expected signature to start with 'sha256=', got %q", sig[:7])
	}
}

func TestSign_DeterministicForSameInput(t *testing.T) {
	payload := []byte(`{"event":"order.created","amount":100}`)
	secret := "supersecret"

	sig1 := signer.Sign(secret, payload)
	sig2 := signer.Sign(secret, payload)

	if sig1 != sig2 {
		t.Errorf("expected same signature for same input; got %q and %q", sig1, sig2)
	}
}

func TestSign_DifferentForDifferentSecret(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	sig1 := signer.Sign("secret-a", payload)
	sig2 := signer.Sign("secret-b", payload)

	if sig1 == sig2 {
		t.Error("expected different signatures for different secrets")
	}
}

func TestSign_DifferentForDifferentPayload(t *testing.T) {
	secret := "mysecret"
	sig1 := signer.Sign(secret, []byte(`{"amount":100}`))
	sig2 := signer.Sign(secret, []byte(`{"amount":200}`))

	if sig1 == sig2 {
		t.Error("expected different signatures for different payloads")
	}
}

func TestVerify_ValidSignature(t *testing.T) {
	secret := "correct-secret"
	payload := []byte(`{"event":"ping"}`)
	sig := signer.Sign(secret, payload)

	if !signer.Verify(secret, payload, sig) {
		t.Error("expected valid signature to pass verification")
	}
}

func TestVerify_InvalidSignature(t *testing.T) {
	payload := []byte(`{"event":"ping"}`)
	sig := signer.Sign("correct-secret", payload)

	if signer.Verify("wrong-secret", payload, sig) {
		t.Error("expected wrong-secret to fail verification")
	}
}

func TestVerify_TamperedPayload(t *testing.T) {
	secret := "mysecret"
	sig := signer.Sign(secret, []byte(`{"amount":100}`))

	if signer.Verify(secret, []byte(`{"amount":999}`), sig) {
		t.Error("expected tampered payload to fail verification")
	}
}
