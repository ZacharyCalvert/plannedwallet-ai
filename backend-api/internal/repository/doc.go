// Package repository defines data access abstractions and persistence implementations
// for the Planned Wallet API.
//
// Key components include:
//   - Repository interface defining CRUD operations for users, financial accounts,
//     and checking records.
//   - PostgresRepository providing production-grade persistence using parameterized SQL queries,
//     unique violation mapping, and query filtering.
//   - MemoryRepository providing a thread-safe in-memory implementation for unit testing
//     and local development fallback.
package repository
