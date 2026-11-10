// Package auth provides authentication, credential verification, and token management
// for the Planned Wallet API.
//
// Key capabilities include:
//   - Bcrypt-based password salting, hashing, and comparison.
//   - JWT access token issuance, signing (HMAC-SHA256), and cryptographic validation.
//   - Gin HTTP middleware for Bearer token validation and user identity propagation
//     into request contexts.
package auth
