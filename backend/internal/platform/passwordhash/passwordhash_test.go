package passwordhash

import "testing"

// Use small params in tests so the suite stays fast; production params are
// set in DefaultParams() and used only at runtime.
func testParams() Params {
	return Params{MemoryKB: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
}

func TestHashAndVerify_RoundTrip(t *testing.T) {
	encoded, err := Hash("correct horse battery staple", testParams())
	if err != nil {
		t.Fatalf("unexpected error hashing: %v", err)
	}
	ok, err := Verify("correct horse battery staple", encoded)
	if err != nil {
		t.Fatalf("unexpected error verifying: %v", err)
	}
	if !ok {
		t.Errorf("expected correct password to verify")
	}
}

func TestVerify_RejectsWrongPassword(t *testing.T) {
	encoded, _ := Hash("correct horse battery staple", testParams())
	ok, err := Verify("wrong password", encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Errorf("expected wrong password to fail verification")
	}
}

func TestHash_ProducesDifferentSaltsEachTime(t *testing.T) {
	a, _ := Hash("same password", testParams())
	b, _ := Hash("same password", testParams())
	if a == b {
		t.Errorf("expected two hashes of the same password to differ (random salt)")
	}
}

func TestVerify_RejectsMalformedHash(t *testing.T) {
	_, err := Verify("anything", "not-a-real-hash")
	if err != ErrInvalidHash {
		t.Fatalf("expected ErrInvalidHash, got %v", err)
	}
}
