// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the PolyForm Shield 1.0.0 license
// that can be found in the licenses directory at the root of this repository, also available at
// https://polyformproject.org/wp-content/uploads/2020/06/PolyForm-Shield-1.0.0.txt.

package delegate

import (
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strings"
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

// Token copied from Harness Saas UI is base64 encoded. However, since kubernetes secret is used to create the token
// with token value put in 'data' field of secret yaml as plain text, the token passed to delegate agent is already
// decoded. For Docker delegates, token passes to delegate agent is still not decoded. This function is to provide
// compatibility to both use cases.
func GetBase64DecodedTokenString(token string) string {
	// Step 1: Check if the token is base64 encoded
	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return token // Not base64, return original token
	}

	// Step 2: Decode the token, check if the decoded result is a hexadecimal string
	decodedStr := strings.TrimSpace(string(decoded))
	if isHexadecimalString(decodedStr) {
		return decodedStr
	}
	return token
}

// A helper function to check if a string is a 32-character hexadecimal string
func isHexadecimalString(decodedToken string) bool {
	match, _ := regexp.MatchString("^[0-9A-Fa-f]{32}$", decodedToken)
	return match
}
