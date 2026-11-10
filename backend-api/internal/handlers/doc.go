// Package handlers implements HTTP request routing, input validation, and RESTful
// endpoint handlers for the Planned Wallet API according to schema.yml.
//
// Handlers cover:
//   - User self-signup and profile registration.
//   - Token exchange and credential verification for JWT authentication.
//   - Financial account creation and type validation against supported account types.
//   - Checking account transaction records CRUD operations (create, list with filters,
//     read by ID, full update, partial patch for clearance status, and deletion).
package handlers
