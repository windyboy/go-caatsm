# Weather Parser Documentation

## Overview

The weather parser module provides parsing capabilities for aviation weather reports including METAR, SPECI, and TAF messages. It is integrated into the system using a composite parser pattern that routes weather reports to the weather parser while maintaining backward compatibility with existing aviation telegram parsing.

## Architecture

The weather parser lives cleanly inside the pure-domain `internal/telegram/weather/` package and integrates with `telegram.Router`:

- **Weather Domain Types & Errors** (`internal/telegram/weather/types.go`): `Message`, `Metar`, `Taf`, weather element structs (`Wind`, `Visibility`, `Cloud`, `WeatherPhenomenon`, etc.), and typed errors
- **Weather Parser Implementation** (`internal/telegram/weather/`): Lexer (`lexer.go`), classifier (`classifier.go`), METAR/SPECI parser (`metar_parser.go`), TAF parser (`taf_parser.go`), shared element parser (`common_elements.go`), normalization (`normalize.go`), and unified `Parser` (`parser.go`)
- **Composite Router** (`internal/telegram/router.go`): `Router` inspects headers via `weather.Classifier` and routes between `weather.Parser` and `aviation.Parser`, converting results to `*telegram.Message`

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
2. `telegram.Router` checks whether the payload header matches a weather report (`METAR`/`SPECI`/`TAF`)
3. If classified as weather: `weather.Parser.Parse(raw)` parses the report and maps it to `*telegram.Message` (`Category` = `"METAR"` / `"SPECI"` / `"TAF"`, `Protocol` = `"WMO"`); any weather parse error is returned directly without falling back to the aviation parser
4. If `aftn.validation_enabled = true` in config, `AFTNValidator` skips weather messages (`Protocol == "WMO"` or weather categories) instead of misapplying AFTN 6-digit `DDHHMM` validation
5. Parsed result stored in `aviation.telegrams` table with `category` = `"METAR"` / `"SPECI"` / `"TAF"`
6. Structured data stored in `body_data` JSONB field

### Database Storage

Weather reports are stored in the existing `telegrams` table:

- `category`: "METAR", "SPECI", or "TAF"
- `body_data`: JSONB containing structured weather data
- `content`: Original raw text
- `message_id`: Generated as `{station}-{issue_time}`

## Testing

Weather parser unit tests are located in `internal/telegram/weather/`, and composite routing tests are in `internal/telegram/`:

- `internal/telegram/weather/classifier_test.go`: Tests report type classification
- `internal/telegram/weather/metar_parser_test.go`: Tests METAR/SPECI parsing
- `internal/telegram/weather/taf_parser_test.go`: Tests TAF parsing
- `internal/telegram/router_test.go`: Tests composite router dispatch and error propagation

Run tests:
```bash
go test ./internal/telegram/... -v
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

