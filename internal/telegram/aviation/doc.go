// Package aviation provides pure-Go parsing and validation for ICAO aviation telegrams.
//
// Supported message categories:
//   - ARR (Arrival)
//   - DEP (Departure)
//   - CNL (Cancellation)
//   - DLA (Delay)
//   - FPL (Flight Plan per ICAO Doc 4444)
//
// The package uses Go's RE2-based regexp engine (guaranteed linear time),
// performs zero external IO or logging, and co-locates each message model
// with its Validate() rules and parser.
package aviation
