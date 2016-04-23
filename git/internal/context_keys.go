package internal

type TokenContextKey struct{}
type ClientContextKey struct{}

// To override http.Client in tests
var HttpClient ClientContextKey
