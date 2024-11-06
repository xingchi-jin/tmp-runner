package gcplogger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/harness/runner/logger"

	"github.com/harness/runner/delegateshell/client"
	"github.com/patrickmn/go-cache"
	"golang.org/x/oauth2"
)

var NoTokenError = errors.New("no log token defined on the server")
var ExpiredError = errors.New("expired token")

const (
	defaultCacheExpirationDuration = 30 * time.Minute
	tokenKey                       = "loggingToken"
)

type TokenManager struct {
	// holds the oauth2 token
	cache         *cache.Cache
	managerClient *client.ManagerClient
	projectID     string
}

func NewTokenManager(ctx context.Context, managerClient *client.ManagerClient) (*TokenManager, error) {
	tokenCache := cache.New(defaultCacheExpirationDuration, -1)
	tokenManager := &TokenManager{
		cache:         tokenCache,
		managerClient: managerClient,
	}
	_, err := tokenManager.setToken(ctx)
	if err != nil {
		return nil, err
	}
	return tokenManager, nil
}

func (tokenManager *TokenManager) Token() (*oauth2.Token, error) {
	token, found := tokenManager.cache.Get(tokenKey)
	if found {
		if token, ok := token.(*oauth2.Token); ok {
			return token, nil
		}
		// If typecast fails, log an error and proceed to refresh
		logger.Errorln("Invalid token type in cache")
	}
	logger.Infoln("Refreshing logging token")

	var err error
	for i := 0; i < 3; i++ {
		token, err = tokenManager.setToken(context.Background())
		if err != nil {
			logger.WithError(err).Warnf("Failed to refresh token, attempt %d of 3", i+1)
		} else if refreshedToken, ok := token.(*oauth2.Token); ok {
			return refreshedToken, nil
		} else {
			logger.Warnf("Invalid token type in cache after refresh, attempt %d of 3", i+1)
		}
		time.Sleep(2 * time.Second)
	}

	logger.Errorln("Cannot refresh logging token after 3 attempts:", err)
	return nil, err
}

func (tokenManager *TokenManager) setToken(ctx context.Context) (*oauth2.Token, error) {
	logCredentials, err := tokenManager.fetchLoggingToken(ctx)
	if err != nil {
		return nil, err
	}
	token := mapToOauthToken(logCredentials)

	if token.AccessToken == "" {
		return nil, NoTokenError
	}
	tokenManager.projectID = logCredentials.ProjectId
	durationUntilExpiration := time.Duration((logCredentials.ExpirationTimeMillis-time.Now().UnixMilli())/2) * time.Millisecond
	tokenManager.cache.Set(tokenKey, token, durationUntilExpiration)
	logger.Printf("Logging token set for: %v", durationUntilExpiration)
	return token, nil
}

func (tokenManager *TokenManager) fetchLoggingToken(ctx context.Context) (*client.AccessTokenBean, error) {
	logCredentials, err := tokenManager.managerClient.GetLoggingToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logger credentials from the server: %w", err)
	}

	if logCredentials == nil || logCredentials.ProjectId == "" || len(logCredentials.TokenValue) == 0 {
		return nil, fmt.Errorf("logging credentials are missing from the server")
	}

	if time.UnixMilli(logCredentials.ExpirationTimeMillis).Before(time.Now()) {
		return nil, ExpiredError
	}

	return logCredentials, nil
}

func mapToOauthToken(credentials *client.AccessTokenBean) *oauth2.Token {
	token := &oauth2.Token{
		AccessToken: credentials.TokenValue,
		Expiry:      time.UnixMilli(credentials.ExpirationTimeMillis),
	}
	return token
}
