package apple

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const appleKeysURL = "https://appleid.apple.com/auth/keys"

type Verifier struct {
	bundleID string

	mu       sync.RWMutex
	keys     map[string]*rsa.PublicKey
	fetchedAt time.Time
}

func NewVerifier(bundleID string) *Verifier {
	return &Verifier{bundleID: bundleID, keys: make(map[string]*rsa.PublicKey)}
}

type AppleClaims struct {
	Email   string `json:"email"`
	Subject string
}

func (v *Verifier) Verify(ctx context.Context, identityToken string) (*AppleClaims, error) {
	token, err := jwt.Parse(identityToken, func(t *jwt.Token) (any, error) {
		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing kid header")
		}
		key, err := v.getKey(ctx, kid)
		if err != nil {
			return nil, err
		}
		return key, nil
	}, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Validate issuer
	iss, _ := claims.GetIssuer()
	if iss != "https://appleid.apple.com" {
		return nil, errors.New("invalid issuer")
	}

	// Validate audience
	aud, _ := claims.GetAudience()
	if !containsString(aud, v.bundleID) {
		return nil, errors.New("invalid audience")
	}

	sub, _ := claims.GetSubject()
	if sub == "" {
		return nil, errors.New("missing subject")
	}

	email, _ := claims["email"].(string)

	return &AppleClaims{
		Email:   email,
		Subject: sub,
	}, nil
}

func (v *Verifier) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	stale := time.Since(v.fetchedAt) > time.Hour
	v.mu.RUnlock()

	if ok && !stale {
		return key, nil
	}

	if err := v.fetchKeys(ctx); err != nil {
		return nil, fmt.Errorf("fetch apple keys: %w", err)
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok = v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("unknown kid: %s", kid)
	}
	return key, nil
}

type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	KID string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (v *Verifier) fetchKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appleKeysURL, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("apple keys endpoint returned %d", resp.StatusCode)
	}

	var set jwks
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return err
	}

	newKeys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, k := range set.Keys {
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return fmt.Errorf("decode n: %w", err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return fmt.Errorf("decode e: %w", err)
		}

		n := new(big.Int).SetBytes(nBytes)
		e := new(big.Int).SetBytes(eBytes)

		newKeys[k.KID] = &rsa.PublicKey{
			N: n,
			E: int(e.Int64()),
		}
	}

	v.mu.Lock()
	v.keys = newKeys
	v.fetchedAt = time.Now()
	v.mu.Unlock()

	return nil
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
