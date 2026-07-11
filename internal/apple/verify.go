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
	"slices"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const appleKeysURL = "https://appleid.apple.com/auth/keys"

// appleKeyFetchTimeout caps the round trip to Apple's JWKS endpoint.
// Without this the auth request would hang on http.DefaultClient (no
// default timeout), pinning a server goroutine if Apple is slow.
const appleKeyFetchTimeout = 5 * time.Second

// jwksCacheTTL is how long we trust cached keys before re-fetching.
// Apple rotates rarely; an hour matches their published guidance.
const jwksCacheTTL = time.Hour

type Verifier struct {
	bundleID string

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time

	// httpClient has an explicit timeout — we never fall through to
	// http.DefaultClient (no timeout) when fetching JWKS.
	httpClient *http.Client
}

func NewVerifier(bundleID string) *Verifier {
	return &Verifier{
		bundleID:   bundleID,
		keys:       make(map[string]*rsa.PublicKey),
		httpClient: &http.Client{Timeout: appleKeyFetchTimeout},
	}
}

type AppleClaims struct {
	Email   string `json:"email"`
	Subject string
}

// Notification is the parsed `events` payload of an Apple
// server-to-server notification JWT.
//
// The Apple JWT body looks like:
//
//	{ "iss":..., "aud": "<bundleID>", "iat":..., "exp":..., "jti":...,
//	  "events": "<a JSON string containing the fields below>" }
//
// We unmarshal the outer JWT, then unmarshal the `events` *string* as a
// second JSON object. See Apple's
// "Receive server-to-server notifications" docs.
type Notification struct {
	Type           string `json:"type"`            // e.g. "consent-revoked", "account-delete"
	Sub            string `json:"sub"`             // Apple user identifier (matches identity-token sub)
	Email          string `json:"email,omitempty"` // present for email-related events
	IsPrivateEmail string `json:"is_private_email,omitempty"`
	EventTime      int64  `json:"event_time"` // ms since epoch
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
	if !slices.Contains(aud, v.bundleID) {
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

// VerifyNotification validates a server-to-server notification JWT from
// Apple and returns the inner event payload.
//
// The signing keys are the same JWKS Apple uses for identity tokens, so we
// reuse the existing fetch/cache. Validation checks:
//  1. RS256 signature against Apple's published keys
//  2. iss == https://appleid.apple.com
//  3. aud == this app's bundle ID
//  4. iat/exp via jwt library's default time validation
//
// The `events` claim is itself a JSON-encoded *string* (not a nested
// object), per Apple's spec — so we decode it twice.
func (v *Verifier) VerifyNotification(ctx context.Context, payload string) (*Notification, error) {
	token, err := jwt.Parse(payload, func(t *jwt.Token) (any, error) {
		kid, ok := t.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing kid header")
		}
		return v.getKey(ctx, kid)
	}, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return nil, fmt.Errorf("parse notification: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid notification token")
	}

	iss, _ := claims.GetIssuer()
	if iss != "https://appleid.apple.com" {
		return nil, errors.New("invalid issuer")
	}
	aud, _ := claims.GetAudience()
	if !slices.Contains(aud, v.bundleID) {
		return nil, errors.New("invalid audience")
	}

	eventsRaw, _ := claims["events"].(string)
	if eventsRaw == "" {
		return nil, errors.New("missing events claim")
	}

	var n Notification
	if err := json.Unmarshal([]byte(eventsRaw), &n); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	if n.Sub == "" {
		return nil, errors.New("events.sub is empty")
	}
	if n.Type == "" {
		return nil, errors.New("events.type is empty")
	}
	return &n, nil
}

func (v *Verifier) getKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	key, ok := v.keys[kid]
	stale := time.Since(v.fetchedAt) > jwksCacheTTL
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
	// Cap the fetch even if the inbound ctx has no deadline. The verifier's
	// own http.Client.Timeout is a backstop in case the caller's ctx never
	// fires (e.g. a long-lived background goroutine).
	ctx, cancel := context.WithTimeout(ctx, appleKeyFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, appleKeysURL, nil)
	if err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
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
