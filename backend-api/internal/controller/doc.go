// Package controller implements the application business logic and controller layer
// in the MVC architecture for the Planned Wallet API.
//
// The controller layer mediates between the HTTP handlers (the view layer) and the
// repository persistence layer. Key responsibilities include:
//   - Enforcing password complexity rules (minimum 10 characters, uppercase, lowercase,
//     digit, and special non-alphanumeric character).
//   - Pre-validating entity existence (e.g. verifying username and email availability
//     prior to creating user records).
//   - Managing token exchange and credential verification with the authentication service.
//   - Enforcing account type constraints and account ownership checks.
//   - Orchestrating checking account transaction records CRUD operations.
package controller
