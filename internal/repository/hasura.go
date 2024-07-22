package repository

import (
	"context"
	"fmt"
	"os"
	"time"

	"caatsm/internal/domain"

	"github.com/hasura/go-graphql-client"
	"golang.org/x/oauth2"
)

type HasuraRepository struct {
	hasuraClient *graphql.Client
}

// NewHasuraClient creates a new HasuraClient
func NewHasuraRepo(endpoint, secret string) *HasuraRepository {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GRAPHQL_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	return &HasuraRepository{
		hasuraClient: graphql.NewClient(endpoint, httpClient),
	}
}

// InsertParsedMessage inserts a new ParsedMessage into the Hasura GraphQL API
func (hr *HasuraRepository) InsertParsedMessage(pm *domain.ParsedMessage) error {
	if hr.hasuraClient == nil {
		return fmt.Errorf("hasuraClient is nil")
	}

	var mutation struct {
		InsertParsedMessages struct {
			Returning []domain.ParsedMessage `json:"returning"`
		} `graphql:"insert_messages_one(object: {message_id: $messageId, date_time: $dateTime, priority_indicator: $priorityIndicator, primary_address: $primaryAddress, secondary_addresses: $secondaryAddresses, originator: $originator, originator_date_time: $originatorDateTime, category: $category, body_and_footer: $bodyAndFooter, body_data: $bodyData, received_at: $receivedAt, parsed_at: $parsedAt, need_dispatch: $needDispatch})"`
	}

	// Define the variables with their types
	variables := map[string]interface{}{
		"messageId":          graphql.String(pm.MessageID),
		"dateTime":           graphql.String(pm.DateTime),
		"priorityIndicator":  graphql.String(pm.PriorityIndicator),
		"primaryAddress":     graphql.String(pm.PrimaryAddress),
		"secondaryAddresses": graphql.String("second"),
		"originator":         graphql.String(pm.Originator),
		"originatorDateTime": graphql.String(pm.OriginatorDateTime),
		"category":           graphql.String(pm.Category),
		"bodyAndFooter":      graphql.String(pm.BodyAndFooter),
		"bodyData":           graphql.String("{}"),
		"receivedAt":         pm.ReceivedAt.Format(time.RFC3339),
		"parsedAt":           pm.ParsedAt.Format(time.RFC3339),
		// "dispatchedAt":       pm.DispatchedAt.Format(time.RFC3339),
		"needDispatch": graphql.Boolean(false),
	}

	ctx := context.Background()
	// Execute the mutation with the variables
	err := hr.hasuraClient.Mutate(ctx, &mutation, variables)
	if err != nil {
		return err
	}

	return nil
}
