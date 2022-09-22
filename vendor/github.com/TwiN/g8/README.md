# g8

![test](https://github.com/TwiN/g8/workflows/test/badge.svg?branch=master) 
[![Go Report Card](https://goreportcard.com/badge/github.com/TwiN/g8)](https://goreportcard.com/report/github.com/TwiN/g8)
[![codecov](https://codecov.io/gh/TwiN/g8/branch/master/graph/badge.svg)](https://codecov.io/gh/TwiN/g8)
[![Go version](https://img.shields.io/github/go-mod/go-version/TwiN/g8.svg)](https://github.com/TwiN/g8)
[![Go Reference](https://pkg.go.dev/badge/github.com/TwiN/g8.svg)](https://pkg.go.dev/github.com/TwiN/g8)
[![Follow TwiN](https://img.shields.io/github/followers/TwiN?label=Follow&style=social)](https://github.com/TwiN)

g8, pronounced gate, is a simple Go library for protecting HTTP handlers.

Tired of constantly re-implementing a security layer for each application? Me too, that's why I made g8.


## Installation
```console
go get -u github.com/TwiN/g8
```


## Usage
Because the entire purpose of g8 is to NOT waste time configuring the layer of security, the primary emphasis is to 
keep it as simple as possible.


### Simple
Just want a simple layer of security without the need for advanced permissions? This configuration is what you're
looking for.

```go
authorizationService := g8.NewAuthorizationService().WithToken("mytoken")
gate := g8.New().WithAuthorizationService(authorizationService)

router := http.NewServeMux()
router.Handle("/unprotected", yourHandler)
router.Handle("/protected", gate.Protect(yourHandler))

http.ListenAndServe(":8080", router)
```

The endpoint `/protected` is now only accessible if you pass the header `Authorization: Bearer mytoken`.

If you use `http.HandleFunc` instead of `http.Handle`, you may use `gate.ProtectFunc(yourHandler)` instead.

If you're not using the `Authorization` header, you can specify a custom token extractor. 
This enables use cases like [Protecting a handler using session cookie](#protecting-a-handler-using-session-cookie)


### Advanced permissions
If you have tokens with more permissions than others, g8's permission system will make managing authorization a breeze.

Rather than registering tokens, think of it as registering clients, the only difference being that clients may be 
configured with permissions while tokens cannot. 

```go
authorizationService := g8.NewAuthorizationService().WithClient(g8.NewClient("mytoken").WithPermission("admin"))
gate := g8.New().WithAuthorizationService(authorizationService)

router := http.NewServeMux()
router.Handle("/unprotected", yourHandler)
router.Handle("/protected-with-admin", gate.ProtectWithPermissions(yourHandler, []string{"admin"}))

http.ListenAndServe(":8080", router)
```

The endpoint `/protected-with-admin` is now only accessible if you pass the header `Authorization: Bearer mytoken`,
because the client with the token `mytoken` has the permission `admin`. Note that the following handler would also be
accessible with that token:
```go
router.Handle("/protected", gate.Protect(yourHandler))
```

To clarify, both clients and tokens have access to handlers that aren't protected with extra permissions, and 
essentially, tokens are registered as clients with no extra permissions in the background.

Creating a token like so:
```go
authorizationService := g8.NewAuthorizationService().WithToken("mytoken")
```
is the equivalent of creating the following client:
```go
authorizationService := g8.NewAuthorizationService().WithClient(g8.NewClient("mytoken"))
```


### With client provider
A client provider's task is to retrieve a Client from an external source (e.g. a database) when provided with a token.
You should use a client provider when you have a lot of tokens and it wouldn't make sense to register all of them using
`AuthorizationService`'s `WithToken`/`WithTokens`/`WithClient`/`WithClients`.

Note that the provider is used as a fallback source. As such, if a token is explicitly registered using one of the 4 
aforementioned functions, the client provider will not be used.

```go
clientProvider := g8.NewClientProvider(func(token string) *g8.Client {
    // We'll assume that the following function calls your database and returns a struct "User" that 
    // has the user's token as well as the permissions granted to said user
    user := database.GetUserByToken(token)
    if user != nil {
        return g8.NewClient(user.Token).WithPermissions(user.Permissions)
    }
    return nil
})
authorizationService := g8.NewAuthorizationService().WithClientProvider(clientProvider)
gate := g8.New().WithAuthorizationService(authorizationService)
```

You can also configure the client provider to cache the output of the function you provide to retrieve clients by token:
```go
clientProvider := g8.NewClientProvider(...).WithCache(ttl, maxSize)
```

Since g8 leverages [TwiN/gocache](https://github.com/TwiN/gocache), you can also use gocache's 
constants for configuring the TTL and the maximum size:
- Setting the TTL to `gocache.NoExpiration` (-1) will disable the TTL. 
- Setting the maximum size to `gocache.NoMaxSize` (0) will disable the maximum cache size

If you're using a TTL and have a lot of tokens (100k+), you may want to use `clientProvider.StartJanitor()` to allow 
the cache to passively delete expired entries. If you have to re-initialize the client provider after the janitor has
been started, make sure to stop the janitor first (`clientProvider.StopJanitor()`). This is because the janitor runs on 
a separate goroutine, thus, if you were to re-create a client provider and re-assign it, the old client provider would 
still exist in memory with the old cache. I'm only specifying this for completeness, because for the overwhelming 
majority of people, the gate will be created on application start and never modified again until the application shuts
down, in which case, you don't even need to worry about stopping the janitor.

To avoid any misunderstandings, using a client provider is not mandatory. If you only have a few tokens and you can load
them on application start, you can just leverage `AuthorizationService`'s `WithToken`/`WithTokens`/`WithClient`/`WithClients`.


## AuthorizationService
As the previous examples may have hinted, there are several ways to create clients. The one thing they have
in common is that they all go through AuthorizationService, which is in charge of both managing clients and determining
whether a request should be blocked or allowed through.

| Function           | Description                                                                                                                      | 
|:-------------------|:---------------------------------------------------------------------------------------------------------------------------------|
| WithToken          | Creates a single static client with no extra permissions                                                                         |
| WithTokens         | Creates a slice of static clients with no extra permissions                                                                      |
| WithClient         | Creates a single static client                                                                                                   |
| WithClients        | Creates a slice of static clients                                                                                                |
| WithClientProvider | Creates a client provider which will allow a fallback to a dynamic source (e.g. to a database) when a static client is not found |

Except for `WithClientProvider`, every functions listed above can be called more than once.
As a result, you may safely perform actions like this:
```go
authorizationService := g8.NewAuthorizationService().
    WithToken("123").
    WithToken("456").
    WithClient(g8.NewClient("789").WithPermission("admin"))
gate := g8.New().WithAuthorizationService(authorizationService)
```

Be aware that g8.Client supports a list of permissions as well. You may call `WithPermission` several times, or call
`WithPermissions` with a slice of permissions instead.


### Permissions
Unlike client permissions, handler permissions are requirements.

A client may have as many permissions as you want, but for said client to have access to a handler protected by
permissions, the client must have all permissions defined by said handler in order to have access to it.

In other words, a client with the permissions `create`, `read`, `update` and `delete` would have access to all of these handlers:
```go
gate := g8.New().WithAuthorizationService(g8.NewAuthorizationService().WithClient(g8.NewClient("mytoken").WithPermissions([]string{"create", "read", "update", "delete"})))
router := http.NewServeMux()
router.Handle("/", gate.Protect(homeHandler)) // equivalent of gate.ProtectWithPermissions(homeHandler, []string{})
router.Handle("/create", gate.ProtectWithPermissions(createHandler, []string{"create"}))
router.Handle("/read", gate.ProtectWithPermissions(readHandler, []string{"read"}))
router.Handle("/update", gate.ProtectWithPermissions(updateHandler, []string{"update"}))
router.Handle("/delete", gate.ProtectWithPermissions(deleteHandler, []string{"delete"}))
router.Handle("/crud", gate.ProtectWithPermissions(crudHandler, []string{"create", "read", "update", "delete"}))
```
But it would not have access to the following handler, because while `mytoken` has the `read` permission, it does not 
have the `backup` permission:
```go
router.Handle("/backup", gate.ProtectWithPermissions(&testHandler{}, []string{"read", "backup"}))
```

If you're using an HTTP library that supports middlewares like [mux](https://github.com/gorilla/mux), you can protect 
an entire group of handlers instead using `gate.Protect` or `gate.PermissionMiddleware()`:
```go
router := mux.NewRouter()

userRouter := router.PathPrefix("/").Subrouter()
userRouter.Use(gate.Protect)
userRouter.HandleFunc("/api/v1/users/me", getUserProfile).Methods("GET")
userRouter.HandleFunc("/api/v1/users/me/friends", getUserFriends).Methods("GET")
userRouter.HandleFunc("/api/v1/users/me/email", updateUserEmail).Methods("PATCH")

adminRouter := router.PathPrefix("/").Subrouter()
adminRouter.Use(gate.PermissionMiddleware("admin"))
adminRouter.HandleFunc("/api/v1/users/{id}/ban", banUserByID).Methods("POST")
adminRouter.HandleFunc("/api/v1/users/{id}/delete", deleteUserByID).Methods("DELETE")
```


## Rate limiting
To add a rate limit of 100 requests per second:
```go
gate := g8.New().WithRateLimit(100)
```


## Accessing the token from the protected handlers
If you need to access the token from the handlers you are protecting with g8, you can retrieve it from the
request context by using the key `g8.TokenContextKey`:
```go
http.Handle("/handle", gate.ProtectFunc(func(w http.ResponseWriter, r *http.Request) {
    token, _ := r.Context().Value(g8.TokenContextKey).(string)
    // ...
}))
```

## Examples
### Protecting a handler using session cookie
If you want to only allow authenticated users to access a handler, you can use a custom token extractor function 
combined with a client provider.

First, we'll create a function to extract the session ID from the session cookie. While a session ID does not 
theoretically refer to a token, g8 uses the term `token` as a blanket term to refer to any string that can be used to
identify a client.
```go
customTokenExtractorFunc := func(request *http.Request) string {
    sessionCookie, err := request.Cookie("session")
    if err != nil {
        return ""
    }
    return sessionCookie.Value
}
```

Next, we need to create a client provider that will validate our token, which refers to the session ID in this case.
```go
clientProvider := g8.NewClientProvider(func(token string) *g8.Client {
    // We'll assume that the following function calls your database and validates whether the session is valid.
    isSessionValid := database.CheckIfSessionIsValid(token)
    if !isSessionValid {
        return nil // Returning nil will cause the gate to return a 401 Unauthorized.
    }
    // You could also retrieve the user and their permissions if you wanted instead, but for this example,
    // all we care about is confirming whether the session is valid or not.
    return g8.NewClient(token)
})
```

Keep in mind that you can get really creative with the client provider above.
For instance, you could refresh the session's expiration time, which will allow the user to stay logged in for 
as long as they're active.

You're also not limited to using something stateful like the example above. You could use a JWT and have your client
provider validate said JWT.

Finally, we can create the authorization service and the gate:
```go
authorizationService := g8.NewAuthorizationService().WithClientProvider(clientProvider)
gate := g8.New().WithAuthorizationService(authorizationService).WithCustomTokenExtractor(customTokenExtractorFunc)
```

If you need to access the token (session ID in this case) from the protected handlers, you can retrieve it from the
request context by using the key `g8.TokenContextKey`:
```go
http.Handle("/handle", gate.ProtectFunc(func(w http.ResponseWriter, r *http.Request) {
    sessionID, _ := r.Context().Value(g8.TokenContextKey).(string)
    // ...
}))
```

### Using a custom header
The logic is the same as the example above:
```go
customTokenExtractorFunc := func(request *http.Request) string {
    return request.Header.Get("X-API-Token")
}

clientProvider := g8.NewClientProvider(func(token string) *g8.Client {
    // We'll assume that the following function calls your database and returns a struct "User" that 
    // has the user's token as well as the permissions granted to said user
    user := database.GetUserByToken(token)
    if user != nil {
        return g8.NewClient(user.Token).WithPermissions(user.Permissions)
    }
    return nil
})
authorizationService := g8.NewAuthorizationService().WithClientProvider(clientProvider)
gate := g8.New().WithAuthorizationService(authorizationService).WithCustomTokenExtractor(customTokenExtractorFunc)
```
