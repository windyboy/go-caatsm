# Weather Parser Documentation

## Overview

The weather parser module provides parsing capabilities for aviation weather reports including METAR, SPECI, and TAF messages. It is integrated into the system using a composite parser pattern that routes weather reports to the weather parser while maintaining backward compatibility with existing aviation telegram parsing.

## Architecture

The weather parser follows Clean Architecture principles:

- **Domain Layer** (`internal/domain/weather/`): Core domain types and interfaces
- **Port Layer** (`internal/port/weather_parser.go`): Parser interface definition
- **Adapter Layer** (`internal/adapter/parser/weather/`): Parser implementation
- **Composite Parser** (`internal/adapter/parser/composite.go`): Routes messages to appropriate parser

## Supported Report Types

### METAR (Aviation Routine Weather Report)
Standard hourly weather observations from airports.

**Example:**
```
METAR KJFK 251200Z 35012KT 10SM FEW020 25/18 Q1013=
```

### SPECI (Aviation Selected Special Weather Report)
Special weather observations issued when conditions change significantly.

**Example:**
```
SPECI KORD 251215Z 27015G25KT 5SM -RA BKN030 OVC050 20/18 A2992=
```

### TAF (Terminal Aerodrome Forecast)
Forecast weather conditions for airports, typically valid for 24-30 hours.

**Example:**
```
TAF KJFK 251200Z 2512/2612 35012KT 10SM FEW020 FM251800 36015KT 10SM SCT030=
```

## Parsed Elements

### Core Elements

- **Station**: 4-letter ICAO airport code
- **Time**: Issue/observation time (DDHHmmZ format)
- **Wind**: Direction, speed, gusts, variable conditions
- **Visibility**: Distance, unit (meters or statute miles), directional visibility
- **Clouds**: Type (FEW/SCT/BKN/OVC/VV), altitude, modifiers (CB/TCU)
- **Temperature/Dewpoint**: Temperature in Celsius
- **Altimeter**: Pressure setting (QNH in hPa or A in inHg)
- **Weather Phenomena**: Intensity, descriptors, weather codes

### TAF-Specific Elements

- **Validity Period**: Forecast valid from/to times
- **Periods**: Main forecast, FM (from), TEMPO (temporary), BECMG (becoming)
- **Probability**: PROB30, PROB40 for uncertain conditions

## Error Handling

The parser uses a lenient approach:

- **Unrecognized tokens**: Recorded in `warnings` array, parsing continues
- **Missing required fields**: Returns appropriate domain errors
- **Invalid format**: Returns `ErrInvalidFormat`

This ensures that partial parsing is possible even when some elements are not recognized.

## Usage

The weather parser is automatically integrated via the composite parser. No special configuration is required.

### Message Flow

1. Raw message received
2. Composite parser checks if message is a weather report
3. If weather report: parsed by weather parser
4. If not: parsed by aviation parser (existing behavior)
5. Parsed result stored in `telegrams` table with `category` = "METAR"/"SPECI"/"TAF"
6. Structured data stored in `body_data` JSONB field

### Database Storage

Weather reports are stored in the existing `telegrams` table:

- `category`: "METAR", "SPECI", or "TAF"
- `body_data`: JSONB containing structured weather data
- `content`: Original raw text
- `message_id`: Generated as `{station}-{issue_time}`

## Testing

Test files are located in `internal/adapter/parser/weather/`:

- `classifier_test.go`: Tests report type classification
- `metar_parser_test.go`: Tests METAR/SPECI parsing
- `taf_parser_test.go`: Tests TAF parsing
- `composite_test.go`: Tests composite parser routing

Run tests:
```bash
go test ./internal/adapter/parser/weather/... -v
```

## Limitations and Future Enhancements

Current implementation covers core METAR/TAF elements. Future enhancements may include:

- Runway Visual Range (RVR) parsing
- More comprehensive weather phenomenon codes
- Enhanced TAF period parsing
- Additional METAR modifiers
- Station metadata integration

## References

- [ICAO Annex 3: Meteorological Service for International Air Navigation](https://www.icao.int/safety/meteorology/pages/annex-3.aspx)
- [WMO Manual on Codes](https://library.wmo.int/index.php?lvl=notice_display&id=13617)

