package g8

import (
	"errors"
	"time"

	"github.com/TwiN/gocache/v2"
)

var (
	// ErrNoExpiration is the error returned by ClientProvider.StartCacheJanitor if there was an attempt to start the
	// janitor despite no expiration being configured.
	// To clarify, this is because the cache janitor is only useful when an expiration is set.
	ErrNoExpiration = errors.New("no point starting the janitor if the TTL is set to not expire")

	// ErrCacheNotInitialized is the error returned by ClientProvider.StartCacheJanitor if there was an attempt to start
	// the janitor despite the cache not having been initialized using ClientProvider.WithCache
	ErrCacheNotInitialized = errors.New("cannot start janitor because cache is not configured")
)

// ClientProvider has the task of retrieving a Client from an external source (e.g. a database) when provided with a
// token. It should be used when you have a lot of tokens, and it wouldn't make sense to register all of them using
// AuthorizationService's WithToken, WithTokens, WithClient or WithClients.
//
// Note that the provider is used as a fallback source. As such, if a token is explicitly registered using one of the 4
// aforementioned functions, the client provider will not be used by the AuthorizationService when a request is made
// with said token. It will, however, be called upon if a token that is not explicitly registered in
// AuthorizationService is sent alongside a request going through the Gate.
//
//     clientProvider := g8.NewClientProvider(func(token string) *g8.Client {
//         // We'll assume that the following function calls your database and returns a struct "User" that
//         // has the user's token as well as the permissions granted to said user
//         user := database.GetUserByToken(token)
//         if user != nil {
//             return g8.NewClient(user.Token).WithPermissions(user.Permissions)
//         }
//         return nil
//     })
//     gate := g8.New().WithAuthorizationService(g8.NewAuthorizationService().WithClientProvider(clientProvider))
//
type ClientProvider struct {
	getClientByTokenFunc func(token string) *Client

	cache *gocache.Cache
	ttl   time.Duration
}

// NewClientProvider creates a ClientProvider
// The parameter that must be passed is a function that the provider will use to retrieve a client by a given token
//
// Example:
//     clientProvider := g8.NewClientProvider(func(token string) *g8.Client {
//         // We'll assume that the following function calls your database and returns a struct "User" that
//         // has the user's token as well as the permissions granted to said user
//         user := database.GetUserByToken(token)
//         if user == nil {
//             return nil
//         }
//         return g8.NewClient(user.Token).WithPermissions(user.Permissions)
//     })
//     gate := g8.New().WithAuthorizationService(g8.NewAuthorizationService().WithClientProvider(clientProvider))
//
func NewClientProvider(getClientByTokenFunc func(token string) *Client) *ClientProvider {
	return &ClientProvider{
		getClientByTokenFunc: getClientByTokenFunc,
	}
}

// WithCache adds cache options to the ClientProvider.
//
// ttl is the time until the cache entry will expire. A TTL of gocache.NoExpiration (-1) means no expiration
// maxSize is the maximum amount of entries that can be in the cache at any given time.
// If a value of gocache.NoMaxSize (0) or less is provided for maxSize, there will be no maximum size.
//
// Example:
//     clientProvider := g8.NewClientProvider(func(token string) *g8.Client {
//         // We'll assume that the following function calls your database and returns a struct "User" that
//         // has the user's token as well as the permissions granted to said user
//         user := database.GetUserByToken(token)
//         if user != nil {
//             return g8.NewClient(user.Token).WithPermissions(user.Permissions)
//         }
//         return nil
//     })
//     gate := g8.New().WithAuthorizationService(g8.NewAuthorizationService().WithClientProvider(clientProvider.WithCache(time.Hour, 70000)))
//
func (provider *ClientProvider) WithCache(ttl time.Duration, maxSize int) *ClientProvider {
	provider.cache = gocache.NewCache().WithEvictionPolicy(gocache.LeastRecentlyUsed).WithMaxSize(maxSize)
	provider.ttl = ttl
	return provider
}

// StartCacheJanitor starts the cache janitor, which passively deletes expired cache entries in the background.
//
// Not really necessary unless you have a lot of clients (100000+).
//
// Even without the janitor, active eviction will still happen (i.e. when GetClientByToken is called, but the cache
// entry for the given token has expired, the cache entry will be automatically deleted and re-fetched from the
// user-defined getClientByTokenFunc)
func (provider *ClientProvider) StartCacheJanitor() error {
	if provider.cache == nil {
		// Can't start the cache janitor if there's no cache
		return ErrCacheNotInitialized
	}
	if provider.ttl != gocache.NoExpiration {
		return provider.cache.StartJanitor()
	}
	return ErrNoExpiration
}

// StopCacheJanitor stops the cache janitor
//
// Not required unless your application initializes multiple providers over the course of its lifecycle.
// In English, that means if you initialize a ClientProvider only once on application start and it stays up
// until your application shuts down, you don't need to call this function.
func (provider *ClientProvider) StopCacheJanitor() {
	if provider.cache != nil {
		provider.cache.StopJanitor()
	}
}

// GetClientByToken retrieves a client by its token through the provided getClientByTokenFunc.
func (provider *ClientProvider) GetClientByToken(token string) *Client {
	if provider.cache == nil {
		return provider.getClientByTokenFunc(token)
	}
	if cachedClient, exists := provider.cache.Get(token); exists {
		if cachedClient == nil {
			return nil
		}
		// Safely typecast the client.
		// Regardless of whether the typecast is successful or not, we return client since it'll be either client or
		// nil. Technically, it should never be nil, but it's better to be safe than sorry.
		client, _ := cachedClient.(*Client)
		return client
	}
	client := provider.getClientByTokenFunc(token)
	provider.cache.SetWithTTL(token, client, provider.ttl)
	return client
}
