# go-caatsm

Civil Aviation Authority Telegram Message Processor

A high-performance, production-ready message processing system for aviation telegrams using Clean Architecture, NATS JetStream, and PostgreSQL.

## Architecture

This project follows Clean Architecture principles with clear separation of concerns:

```
/cmd/main/main.go              # Application entry point
/internal
    /app                       # Application layer (business logic orchestration)
        processor.go           # Message processor
    /domain                    # Domain models (pure Go types, no external dependencies)
        aviation.go            # ParsedMessage and related types
    /adapter                   # Adapter layer (interfaces and implementations)
        /parser                # Message parsing adapters
        /mapper                # Data mapping (domain ↔ infrastructure)
        repository.go          # Repository interface
        publisher.go           # Publisher interface
    /infra                     # Infrastructure layer
        /config                # Configuration management (Koanf)
        /nats                  # NATS JetStream client
        /postgres              # PostgreSQL repository (pgx)
        /log                   # Logging (Zap)
/pkg/di                        # Dependency injection (Wire)
```

## Features

- **Clean Architecture**: Clear separation between domain, application, and infrastructure layers
- **NATS JetStream**: Reliable message streaming with automatic retries and dead-letter queues
- **PostgreSQL**: High-performance data persistence using pgx with batch operations
- **Dependency Injection**: Google Wire for compile-time dependency injection
- **Configuration Management**: Koanf for flexible configuration loading (file + environment variables)
- **Structured Logging**: Zap logger with configurable levels and formats
- **Batch Processing**: Efficient batch message processing and database inserts

## Prerequisites

- Go 1.22+
- PostgreSQL 12+
- NATS Server with JetStream enabled

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd go-caatsm
```

2. Install dependencies:
```bash
go mod download
```

3. Set up PostgreSQL database:
```bash
psql -U postgres -f internal/repository/telegrams.ddl
```

4. Configure the application:
   - Copy `configs/config.dev.toml` and modify as needed
   - Or set environment variables with `CAATSM_` prefix

## Configuration

Configuration is loaded from TOML files and environment variables. The configuration file should be located at `configs/config.{env}.toml` where `{env}` is determined by the `GO_ENV` environment variable (defaults to `dev`).

### Configuration Structure

```toml
[nats]
url = "nats://localhost:4222"
stream = "TELEGRAM"
consumer = "telegram-consumer"

[nats.stream_limits]
max_msgs = 100000
max_bytes = 67108864
max_age = "24h"
discard = "old"
storage = "file"
replicas = 1

[nats.consumer]
max_deliver = 5
ack_wait = "30s"
max_ack_pending = 1024

[subscription]
topic = "telegram.serial"

[publisher]
topic = "telegram.json"

[postgres]
url = "postgres://user:password@localhost:5432/aviation?sslmode=disable"
max_conns = 10
min_conns = 2

[app]
batch_size = 50
batch_timeout = "2s"
monitor_interval = "30s"

[log]
level = "info"
format = "json"
```

### Environment Variables

You can override any configuration value using environment variables with the `CAATSM_` prefix:

```bash
export CAATSM_NATS_URL="nats://nats-server:4222"
export CAATSM_POSTGRES_URL="postgres://user:pass@db:5432/aviation"
export CAATSM_LOG_LEVEL="debug"
```

Environment variable names are converted from `CAATSM_NATS_URL` to `nats.url` in the configuration.

## Usage

### Build

```bash
go build -o bin/receiver ./cmd/main
```

Or using Task:

```bash
task build
```

### Run

```bash
# Development mode
GO_ENV=dev ./bin/receiver listen

# Production mode
GO_ENV=prod ./bin/receiver listen
```

Or using Task:

```bash
task run-dev
```

### Command Line Options

```bash
./bin/receiver listen --help

Flags:
  -n, --nats string      Nats server address (default: "nats://localhost:4222")
  -t, --topic string     Nats topic to listen to (default: "telegram.serial")
```

## Development

### Project Structure

- **Domain Layer** (`internal/domain`): Pure business logic and domain models
- **Application Layer** (`internal/app`): Orchestrates business flows
- **Adapter Layer** (`internal/adapter`): Interfaces and adapters between layers
- **Infrastructure Layer** (`internal/infra`): External concerns (NATS, PostgreSQL, config, logging)

### Adding New Features

1. **Domain Changes**: Add to `internal/domain` (no external dependencies)
2. **Business Logic**: Add to `internal/app`
3. **External Integrations**: Add to `internal/infra`
4. **Adapters**: Add to `internal/adapter` to bridge between layers

### Dependency Injection

Dependencies are managed using Google Wire. To add a new dependency:

1. Create a provider function in the appropriate package
2. Add it to `pkg/di/wire.go`
3. Run `wire ./pkg/di` to regenerate `wire_gen.go`

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
task coverage
```

## Message Flow

1. **NATS Consumer** receives raw telegram messages from JetStream
2. **MessageProcessor** orchestrates the processing:
   - Parses the message using the Parser adapter
   - Stores the parsed message in PostgreSQL via Repository
   - Publishes the parsed message to the output topic via Publisher
3. **ACK/NAK** is sent based on processing success/failure
4. **Retry Logic** handles transient failures automatically

## Message Parsing

The system supports parsing of aviation telegram messages in the standard ICAO format. All messages follow a common header structure, followed by a message body that varies by message type.

### Message Format

All telegrams follow this general structure:

```
ZCZC <MessageID> <DateTime>
<PriorityIndicator> <PrimaryAddress>
<SecondaryAddresses>
<Originator>
<Body>
NNNN
```

**Header Fields:**
- `ZCZC`: Start indicator
- `MessageID`: Unique message identifier (e.g., "TMQ1324")
- `DateTime`: Message date and time (e.g., "150631")
- `PriorityIndicator`: Message priority (e.g., "FF", "DD")
- `PrimaryAddress`: Primary recipient address (ICAO code)
- `SecondaryAddresses`: Additional recipient addresses
- `Originator`: Message originator (optional)

### Supported Message Types

The parser supports the following message categories:

#### 1. ARR - Arrival Message

Arrival messages report aircraft arrival information.

**Format:**
```
(ARR-<FlightNumber>[/<SSR>]-<DepartureAirport>-<ArrivalAirport><ArrivalTime>)
```

**Parsed Fields:**
- `category`: "ARR"
- `aircraft_id`: Aircraft identification/flight number
- `ssr_mode_and_code`: SSR mode and code (optional)
- `departure_airport`: Departure airport ICAO code
- `departure_time`: Departure time
- `arrival_airport`: Arrival airport ICAO code
- `arrival_time`: Arrival time
- `estimated_elapsed_time`: Estimated flight duration (optional)
- `alternate_airport`: Alternate airport (optional)
- `other_info`: Additional information (optional)

**Example:**
```
ZCZC ARR1234 150631
FF ZBTJZPZX
150630 ZBACZQZX
(ARR-CCA1234-A1234-ZBTJ1500-ZGGG0135)
NNNN
```

#### 2. DEP - Departure Message

Departure messages report aircraft departure information.

**Format:**
```
(DEP-<FlightNumber>[/<SSR>]-<DepartureAirport><DepartureTime>-<Destination>)
```

**Parsed Fields:**
- `category`: "DEP"
- `aircraft_id`: Aircraft identification/flight number
- `ssr_mode_and_code`: SSR mode and code (optional)
- `departure_airport`: Departure airport ICAO code
- `departure_time`: Departure time
- `destination`: Destination airport ICAO code
- `estimated_elapsed_time`: Estimated flight duration
- `alternate_airport`: Alternate airport (optional)
- `other_info`: Additional information (optional)

**Example:**
```
ZCZC DEP5678 120915
DD KLAXZPZX
120914 KSFOZQZX
(DEP-ABC5678-A1234-ZBTJ1440-ZGGG)
NNNN
```

#### 3. CNL - Cancellation Message

Cancellation messages indicate flight cancellations.

**Format:**
```
(CNL-<FlightNumber>-<DepartureAirport>-<DestinationAirport>)
```

**Parsed Fields:**
- `category`: "CNL"
- `aircraft_id`: Aircraft identification/flight number
- `departure_airport`: Departure airport ICAO code
- `destination_airport`: Destination airport ICAO code
- `other_info`: Additional information (optional)

**Example:**
```
ZCZC CNL9012 150631
FF ZBTJZPZX
(CNL-CCA9012-ZBTJ-ZGGG)
NNNN
```

#### 4. DLA - Delay Message

Delay messages report flight delays with new departure times.

**Format:**
```
(DLA-<FlightNumber>[/<SSR>]-<DepartureAirport>[<NewDepartureTime>]-<ArrivalAirport>[<ArrivalTime>])
```

**Parsed Fields:**
- `category`: "DLA"
- `aircraft_id`: Aircraft identification/flight number
- `ssr_mode_and_code`: SSR mode and code (optional)
- `departure_airport`: Departure airport ICAO code
- `new_departure_time`: New departure time (optional)
- `arrival_airport`: Arrival airport ICAO code
- `arrival_time`: Estimated arrival time (optional)
- `other_info`: Additional information (optional)

**Example:**
```
ZCZC DLA3456 150631
FF ZBTJZPZX
(DLA-CCA3456-A1234-ZBTJ1600-ZGGG0200)
NNNN
```

#### 5. FPL - Flight Plan Message

Flight plan messages contain detailed flight planning information.

**Format:**
```
(FPL-<FlightNumber>-<Indicator>
-<AircraftID>/<SSR>
-<DepartureAirport><DepartureTime>
-<Speed><Level> <Route>
-<Destination><EstimatedTime> <AlternateAirport>
-<OtherInfo>)
```

**Parsed Fields:**
- `category`: "FPL"
- `flight_number`: Flight number
- `reference_data`: Reference data (optional)
- `aircraft_id`: Aircraft identification
- `ssr_mode_and_code`: SSR mode and code
- `flight_rules_and_type`: Flight rules and type
- `cruising_speed_and_level`: Cruising speed and flight level
- `departure_airport`: Departure airport ICAO code
- `departure_time`: Departure time
- `route`: Flight route
- `destination_and_total_time`: Destination and estimated total time
- `alternate_airport`: Alternate airport (optional)
- `estimated_arrival_time`: Estimated arrival time
- `pbn`: Performance-based navigation equipment
- `navigation_equipment`: Navigation equipment
- `estimated_elapsed_time`: Estimated elapsed time
- `selcal_code`: SELCAL code
- `register`: Aircraft registration (optional)
- `performance_category`: Performance category
- `reroute_information`: Reroute information (optional)
- `remarks`: Remarks (optional)

**Example:**
```
ZCZC FPL7890 150631
FF ZBTJZPZX
(FPL-JAE7433-IS
-B744/H-SXIRPZJWY/S
-ZBTJ1755
-K0926S0920 CG A326 VYK W80 HUR B339 GM A575 MANSA/K0919S0980
-EDDF0948 EDDK
-EET/ZMUB0100 UNKL0236
REG/B2422 SEL/JLAD
NAV/RNAV1 RNAV5 RNP4
RMK/AGCS EQUIPPED)
NNNN
```

### Parsing Process

1. **Header Parsing**: The parser extracts header information including message ID, date/time, priority, and addresses
2. **Category Detection**: The parser identifies the message category from the body content
3. **Body Parsing**: Based on the category, the parser applies the appropriate regex pattern to extract structured data
4. **Data Mapping**: Extracted data is mapped to domain model structures (ARR, DEP, CNL, DLA, or FPL)
5. **Storage**: The parsed message is stored in PostgreSQL with:
   - Raw content in `content` field
   - Parsed structured data in `body_data` field (JSONB)
   - Metadata in dedicated columns

### Parsed Message Structure

All parsed messages are stored in the `ParsedMessage` domain model:

```go
type ParsedMessage struct {
    Uuid               string      // Unique identifier
    MessageID          string      // Telegram message ID
    DateTime           string      // Message date/time
    PriorityIndicator  string      // Priority level
    PrimaryAddress     string      // Primary recipient
    SecondaryAddresses string      // Secondary recipients
    Originator         string      // Message originator
    OriginatorDateTime string      // Originator date/time
    Category           string      // Message category (ARR, DEP, CNL, DLA, FPL)
    Content            string      // Raw message content
    BodyData           interface{} // Parsed body data (ARR, DEP, CNL, DLA, or FPL struct)
    ReceivedAt         time.Time   // Reception timestamp
    ParsedAt           time.Time   // Parsing timestamp
    DispatchedAt       time.Time   // Dispatch timestamp
    NeedDispatch       bool        // Dispatch flag
    Parsed             bool        // Parsing success flag
    Comments           string      // Parsing comments/errors
}
```

### Error Handling

- **Invalid Format**: Messages that don't match expected formats are stored with `Parsed = false` and error details in `Comments`
- **Partial Parsing**: Header parsing failures result in storing raw content only
- **Category Mismatch**: Unsupported categories are logged and stored with parsing errors

## Database Schema

The application uses the `aviation.telegrams` table. See `internal/repository/telegrams.ddl` for the schema definition.

Key fields:
- `uuid`: Primary key (UUID)
- `message_id`: Telegram message ID
- `content`: Raw message content
- `body_data`: Parsed body data (JSONB)
- `received_at`, `parsed_at`, `dispatched_at`: Timestamps

## Logging

Logging uses Zap with structured logging. Log levels and format can be configured:

- **Levels**: `debug`, `info`, `warn`, `error`
- **Formats**: `json` (production) or `console` (development)

Logs include contextual information:
- Message IDs
- Subject names
- Stream names
- Processing attempts

## Performance

- **Batch Processing**: Messages are processed in configurable batches (default: 50)
- **Database Inserts**: Uses PostgreSQL `COPY FROM` for efficient batch inserts
- **Connection Pooling**: Configurable PostgreSQL connection pool
- **JetStream**: Reliable message delivery with automatic retries

## Troubleshooting

### Connection Issues

- **NATS**: Check that NATS server is running and JetStream is enabled
- **PostgreSQL**: Verify database connection string and that the schema exists

### Message Processing Issues

- Check logs for parsing errors
- Verify message format matches expected telegram format
- Check database constraints and indexes

### Configuration Issues

- Ensure `GO_ENV` is set correctly
- Verify configuration file exists at `configs/config.{env}.toml`
- Check environment variable names use `CAATSM_` prefix

## Migration from Legacy System

This project was refactored from:
- **Watermill** → **nats.go JetStream**
- **Hasura GraphQL** → **PostgreSQL pgx**
- **Viper** → **Koanf**
- **Manual DI** → **Google Wire**

The legacy code has been removed. See the project history for migration details.

## License

[Add your license here]

## Contributing

[Add contributing guidelines here]

