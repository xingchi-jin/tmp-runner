// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package delegate

import (
	"context"
	"time"

	"github.com/harness/runner/logger"
	"github.com/harness/runner/utils"
	"github.com/patrickmn/go-cache"
)

var (
	audience       = "audience"
	issuer         = "issuer"
	expirationTime = 20 * time.Minute
)

type TokenCache struct {
	id         string
	secret     string
	secretHash string
	expiry     time.Duration
	c          *cache.Cache
}

// NewTokenCache creates a token cache which creates a new token
// after the expiry time is over
func NewTokenCache(id, secret string) *TokenCache {
	// purge expired tokens from the cache at expirationTime/3 intervals
	c := cache.New(cache.DefaultExpiration, expirationTime/3)
	return &TokenCache{
		id:         id,
		secret:     secret,
		secretHash: utils.HashSHA256(secret),
		expiry:     expirationTime,
		c:          c,
	}
}

// Get returns the value of the account token.
// If the token is cached, it returns from there. Otherwise
// it creates a new token with a new expiration time.
func (t *TokenCache) Get(ctx context.Context) (string, error) {
	tv, found := t.c.Get(t.id)
	if found {
		return tv.(string), nil
	}
	logger.WithField(ctx, "id", t.id).Infoln("refreshing token")
	token, err := Token(audience, issuer, t.id, t.secret, t.expiry)
	if err != nil {
		return "", err
	}
	// refresh token before the expiration time to give some buffer
	t.c.Set(t.id, token, t.expiry/2)
	return token, nil
}

func (t *TokenCache) GetTokenHash() string {
	return t.secretHash
}
