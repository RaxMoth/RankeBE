package apple

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// These tests stand up a fake JWKS endpoint (mirroring
// https://appleid.apple.com/auth/keys) and mint identity / notification
// tokens with a locally generated RSA key, so the full parse → signature →
// claim-validation path runs without ever touching Apple.

const (
	testBundleID = "com.ranke.app"
	testKID      = "test-kid-1"
	appleIssuer  = "https://appleid.apple.com"
)

// newTestVerifier returns a Verifier wired to a local JWKS server that serves
// the public half of key, plus the private key for minting tokens and a
// pointer to the JWKS request counter (for cache assertions).
func newTestVerifier(t *testing.T) (*Verifier, *rsa.PrivateKey, *int32) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks{Keys: []jwk{{
			KID: testKID,
			N:   base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
			E:   base64.RawURLEncoding.EncodeToString(encodeExponent(key.PublicKey.E)),
		}}})
	}))
	t.Cleanup(srv.Close)

	v := NewVerifier(testBundleID)
	v.keysURL = srv.URL
	return v, key, &hits
}

// encodeExponent encodes an int exponent as a big-endian byte slice, matching
// how Apple publishes the "e" JWK parameter.
func encodeExponent(e int) []byte {
	var b []byte
	for e > 0 {
		b = append([]byte{byte(e & 0xff)}, b...)
		e >>= 8
	}
	return b
}

// signToken mints an RS256 JWT with the test kid header.
func signToken(t *testing.T, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKID
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

func validIdentityClaims() jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss":   appleIssuer,
		"aud":   testBundleID,
		"sub":   "001234.abcd",
		"email": "user@example.com",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
}

func TestVerify_Valid(t *testing.T) {
	v, key, _ := newTestVerifier(t)
	token := signToken(t, key, validIdentityClaims())

	claims, err := v.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Subject != "001234.abcd" {
		t.Errorf("subject = %q, want 001234.abcd", claims.Subject)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("email = %q, want user@example.com", claims.Email)
	}
}

func TestVerify_Rejects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(jwt.MapClaims)
	}{
		{"wrong audience", func(c jwt.MapClaims) { c["aud"] = "com.evil.app" }},
		{"wrong issuer", func(c jwt.MapClaims) { c["iss"] = "https://evil.example.com" }},
		{"expired", func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Hour).Unix() }},
		{"missing subject", func(c jwt.MapClaims) { delete(c, "sub") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, key, _ := newTestVerifier(t)
			claims := validIdentityClaims()
			tt.mutate(claims)
			token := signToken(t, key, claims)

			if _, err := v.Verify(context.Background(), token); err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
		})
	}
}

func TestVerify_WrongSigningKey(t *testing.T) {
	v, _, _ := newTestVerifier(t)

	// Sign with a *different* key than the JWKS publishes.
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	token := signToken(t, other, validIdentityClaims())

	if _, err := v.Verify(context.Background(), token); err == nil {
		t.Fatal("expected signature-verification failure, got nil")
	}
}

func TestVerify_RejectsNoneAlg(t *testing.T) {
	v, _, _ := newTestVerifier(t)

	// The "alg: none" downgrade attack — jwt.WithValidMethods([]string{"RS256"})
	// must reject it before any key lookup.
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, validIdentityClaims())
	tok.Header["kid"] = testKID
	token, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none: %v", err)
	}

	if _, err := v.Verify(context.Background(), token); err == nil {
		t.Fatal("expected alg:none rejection, got nil")
	}
}

func TestVerify_UnknownKID(t *testing.T) {
	v, key, _ := newTestVerifier(t)
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, validIdentityClaims())
	tok.Header["kid"] = "some-other-kid"
	token, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := v.Verify(context.Background(), token); err == nil {
		t.Fatal("expected unknown-kid failure, got nil")
	}
}

func TestVerify_CachesJWKS(t *testing.T) {
	v, key, hits := newTestVerifier(t)
	token := signToken(t, key, validIdentityClaims())

	for i := 0; i < 3; i++ {
		if _, err := v.Verify(context.Background(), token); err != nil {
			t.Fatalf("Verify #%d: %v", i, err)
		}
	}
	if got := atomic.LoadInt32(hits); got != 1 {
		t.Errorf("JWKS fetched %d times, want 1 (cached)", got)
	}
}

func validNotificationClaims(t *testing.T, event Notification) jwt.MapClaims {
	t.Helper()
	events, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	now := time.Now()
	return jwt.MapClaims{
		"iss":    appleIssuer,
		"aud":    testBundleID,
		"iat":    now.Unix(),
		"exp":    now.Add(time.Hour).Unix(),
		"events": string(events),
	}
}

func TestVerifyNotification_Valid(t *testing.T) {
	v, key, _ := newTestVerifier(t)
	claims := validNotificationClaims(t, Notification{
		Type:      "consent-revoked",
		Sub:       "001234.abcd",
		EventTime: 1700000000000,
	})
	token := signToken(t, key, claims)

	n, err := v.VerifyNotification(context.Background(), token)
	if err != nil {
		t.Fatalf("VerifyNotification: %v", err)
	}
	if n.Type != "consent-revoked" {
		t.Errorf("type = %q, want consent-revoked", n.Type)
	}
	if n.Sub != "001234.abcd" {
		t.Errorf("sub = %q, want 001234.abcd", n.Sub)
	}
}

func TestVerifyNotification_Rejects(t *testing.T) {
	tests := []struct {
		name   string
		claims func(*testing.T) jwt.MapClaims
	}{
		{"wrong audience", func(t *testing.T) jwt.MapClaims {
			c := validNotificationClaims(t, Notification{Type: "account-delete", Sub: "x"})
			c["aud"] = "com.evil.app"
			return c
		}},
		{"missing events", func(t *testing.T) jwt.MapClaims {
			c := validNotificationClaims(t, Notification{Type: "account-delete", Sub: "x"})
			delete(c, "events")
			return c
		}},
		{"empty sub", func(t *testing.T) jwt.MapClaims {
			return validNotificationClaims(t, Notification{Type: "account-delete"})
		}},
		{"empty type", func(t *testing.T) jwt.MapClaims {
			return validNotificationClaims(t, Notification{Sub: "x"})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, key, _ := newTestVerifier(t)
			token := signToken(t, key, tt.claims(t))
			if _, err := v.VerifyNotification(context.Background(), token); err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
		})
	}
}
