// Package database manages PostgreSQL connectivity, connection pooling, and schema initialization
// for the Planned Wallet API.
//
// Key capabilities include:
//   - Connection pool lifecycle configuration, ping healthchecks, and timeout handling.
//   - Automated database schema initialization (InitDatabase) using embedded init.sql DDL
//     for tables including users, accounts, checking records, and asset values, along with foreign keys
//     and performance indexes.
package database
