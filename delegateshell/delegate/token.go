// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package delegate

import (
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// Token generates a token with the given expiry to interact with the Harness manager
func Token(audience, issuer, subject, secret string, expiry time.Duration) (string, error) {
	bytes, err := hex.DecodeString(secret)
	if err != nil {
		return "", err
	}

	enc, err := jose.NewEncrypter(
		jose.A128GCM,
		jose.Recipient{Algorithm: jose.DIRECT, Key: bytes},
		(&jose.EncrypterOptions{}).WithType("JWT"),
	)
	if err != nil {
		return "", err
	}

	cl := jwt.Claims{
		Subject:  subject,
		Issuer:   issuer,
		Audience: []string{audience},
		Expiry:   jwt.NewNumericDate(time.Now().Add(expiry)),
		IssuedAt: jwt.NewNumericDate(time.Now()),
		ID:       uuid.New().String(),
	}
	raw, err := jwt.Encrypted(enc).Claims(cl).CompactSerialize()
	if err != nil {
		return "", err
	}

	return raw, nil
}
