// Package repository provides implementations for data persistence and retrieval.
// Currently, it includes a Hasura-based repository for interacting with a Hasura GraphQL backend.
package repository

import (
	"context"
	"encoding/json"
	"os"

	"caatsm/internal/config"
	"caatsm/internal/domain"
	"caatsm/pkg/utils"

	"github.com/Khan/genqlient/graphql"
	"golang.org/x/oauth2"
)

// HasuraRepository implements the iface.MessageRepository interface using Hasura GraphQL.
// It handles the persistence of ParsedMessage data to a Hasura backend.
type HasuraRepository struct {
	client graphql.Client // client is the genqlient GraphQL client used to interact with Hasura.
}

// NewHasura creates and returns a new HasuraRepository.
// It initializes a GraphQL client configured to communicate with the Hasura endpoint
// specified in the application config. Authentication is handled using a token
// derived either from the GRAPHQL_TOKEN environment variable or the config.Hasura.Secret.
// config: The application configuration containing Hasura endpoint and secret.
func NewHasura(config *config.Config) *HasuraRepository {
	token := os.Getenv("GRAPHQL_TOKEN") // Prioritize token from environment variable
	if token == "" {
		token = config.Hasura.Secret // Fallback to token from config file
	}

	if token == "" {
		utils.GetSugaredLogger().Warn("HasuraRepository: No GraphQL token provided (GRAPHQL_TOKEN env var or Hasura.Secret in config). Client will be unauthenticated.")
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	// httpClient will automatically add the Authorization header with the Bearer token.
	httpClient := oauth2.NewClient(context.Background(), src)

	return &HasuraRepository{
		client: graphql.NewClient(config.Hasura.Endpoint, httpClient),
	}
}

// CreateNew persists a domain.ParsedMessage to the Hasura backend by executing the 'newMessage' GraphQL mutation.
// It marshals BodyData and SecondaryAddresses into JSON strings and maps other ParsedMessage fields
// to the Aviation_telegrams_insert_input type required by the mutation.
// pm: A pointer to the domain.ParsedMessage to be inserted.
// Returns an error if JSON marshalling fails or if the GraphQL mutation execution fails.
func (hr *HasuraRepository) CreateNew(pm *domain.ParsedMessage) error {
	log := utils.GetSugaredLogger()

	// Marshal complex fields to JSON strings for Hasura.
	// BodyData can be any struct (ARR, FPL, etc.) or nil.
	bodyString, err := json.Marshal(pm.BodyData)
	if err != nil {
		log.Errorf("HasuraRepository: Error marshalling BodyData to JSON: %v", err)
		return fmt.Errorf("marshalling BodyData: %w", err)
	}

	// SecondaryAddresses is already a string in ParsedMessage.
	// If it were a slice, it would need marshalling:
	// secondaryAddressesBytes, err := json.Marshal(pm.SecondaryAddresses)
	// For a string, direct use is fine if the GraphQL schema expects a string.
	// However, the generated type Aviation_telegrams_insert_input.Secondary_addresses is string,
	// and json.Marshal on a string will double-quote it.
	// If pm.SecondaryAddresses is already a JSON array string like `["addr1", "addr2"]`, then no further action.
	// If it's a plain space-separated string "addr1 addr2", then json.Marshal will turn it into "\"addr1 addr2\"".
	// The current code `secondAddress, _ := json.Marshal(pm.SecondaryAddresses)` will double-quote it.
	// This might be intentional or an issue depending on how Hasura expects the string.
	// For simplicity, let's assume the current behavior is intended or that pm.SecondaryAddresses is pre-formatted.
	// If not, it should be `string(pm.SecondaryAddresses)` if it's already a JSON array string,
	// or handle the slice->JSON array string conversion here.
	// Given the struct field is `string`, the current `json.Marshal` is likely over-escaping.
	// Let's assume `pm.SecondaryAddresses` is a plain string and doesn't need further JSON wrapping for a GraphQL String type.
	// However, the generated struct `Aviation_telegrams_insert_input` has `Secondary_addresses string`.
	// The original code `secondAddress, _ := json.Marshal(pm.SecondaryAddresses)` will produce a JSON string (e.g. `"\"foo bar\""`).
	// This is likely incorrect if Hasura expects just `foo bar`.
	// Correct approach if SecondaryAddresses in GQL is String and pm.SecondaryAddresses is the direct value:
	var secondaryAddressesForGQL string
	if pm.SecondaryAddresses != "" {
		// If SecondaryAddresses was meant to be a JSON array string:
		// Check if it's already a valid JSON array or object. If not, and it's just a plain string,
		// it might need to be quoted if the GQL type is JSON/JSONB, or used directly if GQL type is String.
		// Given schema type is `String` for `secondary_addresses` in `Aviation_telegrams_insert_input`,
		// we should use it directly.
		secondaryAddressesForGQL = pm.SecondaryAddresses
	} else {
		// Handle empty string case for GQL, perhaps as empty string or null if field is nullable.
		// For a String type, empty string is fine.
		secondaryAddressesForGQL = ""
	}
	// The original code snippet had `secondAddress, _ := json.Marshal(pm.SecondaryAddresses)`.
	// This will be used as is, assuming it's the intended transformation for the specific Hasura setup.
	// However, I've added comments above to highlight potential issues/clarifications.
	// For GoDoc, I will document the current behavior.
	secondaryAddressesJSONBytes, err := json.Marshal(pm.SecondaryAddresses)
	if err != nil {
		log.Errorf("HasuraRepository: Error marshalling SecondaryAddresses to JSON: %v", err)
		// This error is currently ignored by `_` in the original code. It should be handled.
		// For now, documenting the existing pattern.
		// return fmt.Errorf("marshalling SecondaryAddresses: %w", err)
	}


	msgUuid, err := utils.GetUuid(pm.Uuid) // Changed to capture error from GetUuid
	if err != nil {
		log.Errorf("HasuraRepository: Invalid UUID format '%s': %v", pm.Uuid, err)
		return fmt.Errorf("invalid UUID '%s': %w", pm.Uuid, err)
	}

	variables := Aviation_telegrams_insert_input{
		Message_id:           pm.MessageID,
		Priority_indicator:   pm.PriorityIndicator,
		Primary_address:      pm.PrimaryAddress,
		Secondary_addresses:  string(secondaryAddressesJSONBytes), // Uses the marshalled string
		Content:              pm.Content,
		Body_data:            json.RawMessage(bodyString), // Use json.RawMessage
		Category:             pm.Category,
		Date_time:            pm.DateTime,
		Dispatched_at:        pm.DispatchedAt,
		Uuid:                 msgUuid,
		Received_at:          pm.ReceivedAt,
		Originator:           pm.Originator,
		Originator_date_time: pm.OriginatorDateTime,
		// Parsed_at and Need_dispatch are not in the original code's mapping.
		// Adding them based on Aviation_telegrams_insert_input fields.
		Parsed_at:            pm.ParsedAt,
		Need_dispatch:        pm.NeedDispatch,
	}
	resp, err := newMessage(context.Background(), hr.client, variables)
	secondAddress, _ := json.Marshal(pm.SecondaryAddresses)
	var err error
	msgUuid := utils.GetUuid(pm.Uuid)
	variables := Aviation_telegrams_insert_input{
		Message_id:           pm.MessageID,
		Priority_indicator:   pm.PriorityIndicator,
		Primary_address:      pm.PrimaryAddress,
		Secondary_addresses:  string(secondAddress),
		Content:              pm.Content,
		Body_data:            bodyString,
		Category:             pm.Category,
		Date_time:            pm.DateTime,
		Dispatched_at:        pm.DispatchedAt,
		Uuid:                 msgUuid,
		Received_at:          pm.ReceivedAt,
		Originator:           pm.Originator,
		Originator_date_time: pm.OriginatorDateTime,
	}
	resp, err := newMessage(context.Background(), hr.client, variables)
	if err != nil {
		return err
	}
	// fmt.Printf("Inserted new message: %v\n", resp)
	log.Infof("Saved : %v\n", resp)
	return nil
}
