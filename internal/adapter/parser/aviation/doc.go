// Package aviation provides parsing capabilities for ICAO aviation telegrams.
//
// This package implements a robust parser for ICAO-format aviation messages
// following AFTN (Aeronautical Fixed Telecommunication Network) standards.
// It supports multiple message types used in civil aviation operations.
//
// # Supported Message Types
//
// The parser handles five primary message categories:
//
//   - ARR (Arrival): Aircraft arrival notifications with departure/arrival airports and times
//   - DEP (Departure): Aircraft departure notifications with departure/destination airports
//   - CNL (Cancellation): Flight cancellation messages
//   - DLA (Delay): Flight delay notifications with updated times
//   - FPL (Flight Plan): Complete flight plan messages per ICAO Doc 4444
//
// # Architecture
//
// The parser uses a registry-based architecture with specialized parsers for each
// message category. The main components are:
//
//   - Parse(): Entry point for parsing complete telegrams (header + body)
//   - ParseHeader(): Extracts header metadata (addresses, originator, timestamps)
//   - BodyParser: Routes body content to category-specific parsers
//   - CategoryParser: Interface implemented by each message type parser
//
// # Security Features
//
// The parser includes multiple security protections:
//
//   - Input size validation (max 1800 chars per AFTN standard)
//   - ReDoS protection with 100ms regex timeout
//   - Field validation to prevent nil pointer dereferences
//   - Error message sanitization to prevent data leakage
//
// # Usage Example
//
//	rawTelegram := `ZCZC ABC123 261530
//	FF ZBBBZPZX
//	261530 ZBBBYMYX
//	(ARR-CES5470/A1234-ZBTJ-ZSHC1614)
//	NNNN`
//
//	parsed, err := aviation.Parse(rawTelegram)
//	if err != nil {
//	    // Parse failed - check parsed.Status for error type
//	    log.Printf("Parse error: %v, status: %s", err, parsed.Status)
//	    return
//	}
//
//	// Parse succeeded - access structured data
//	if arr, ok := parsed.BodyData.(*domain.ARR); ok {
//	    fmt.Printf("Flight %s arrived at %s\n", arr.AircraftID, arr.ArrivalAirport)
//	}
//
// # Error Handling
//
// The Parse() function follows a unique error handling pattern: it ALWAYS returns
// a non-nil ParsedTelegram, even when an error occurs. This allows callers to
// persist failed parse attempts with error details for audit and compliance.
//
// Error categories:
//
//   - MessageStatusHeaderError: Invalid header format or size validation failure
//   - MessageStatusBodyError: Invalid body format or unsupported message type
//   - MessageStatusParsed: Successful parse
//
// Parser failures are considered permanent (should be ACK'd in message queue systems).
// The returned ParsedTelegram contains error details in the ErrorReason field.
//
// # Performance
//
// The parser is designed for high-throughput message processing:
//
//   - Zero-allocation tokenization where possible
//   - Compiled regex patterns (initialized once at startup)
//   - No global state or locks (thread-safe by design)
//   - Batch-friendly (no shared mutable state between Parse() calls)
//
// # Standards Compliance
//
// This implementation follows:
//
//   - ICAO Doc 4444 (PANS-ATM) for flight plan format
//   - ICAO Annex 10 for AFTN message structure
//   - AFTN size limits (1800 characters maximum)
//
package aviation
