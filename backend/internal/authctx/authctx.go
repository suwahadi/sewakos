// Package authctx holds the one value every [auth] feature package needs
// from its router: who's making the request. It exists so those packages
// (account, notification, and the ones B3+ add) don't need to import
// internal/server for server.AuthUser — server mounts their routes, so the
// reverse import would be a cycle.
package authctx

// User is the authenticated caller a request handler acts as.
type User struct {
	UserID  int64
	TokenID int64
	Role    string
}
