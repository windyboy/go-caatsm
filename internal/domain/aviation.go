package domain

/*
ZCZC TMQ1324 150631
FF ZBTJZPZX
150630 ZBACZQZX
*/
/*
Start Indicator: ZCZC
Message ID: TMQ1324
Date-Time: 150631
Priority Indicator: FF
Primary Address: ZBTJZPZX
Secondary Addresses: 150630 ZBACZQZX
Originator and Originator Date-Time: Not present in this example but handled if present.
*/
// ParsedMessage holds the parsed data from an aviation message
type ParsedMessage struct {
	StartIndicator     string   // Start of the message indicator
	MessageID          string   // Unique identifier for the message
	DateTime           string   // Date and time the message was created
	PriorityIndicator  string   // Priority level of the message
	PrimaryAddress     string   // Primary recipient address
	SecondaryAddresses []string // Additional recipient addresses
	Originator         string   // Sender of the message
	OriginatorDateTime string   // Date and time when the originator sent the message
	BodyAndFooter      string
}
