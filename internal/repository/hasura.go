package repository

import (
	"context"
	"fmt"
	"os"

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

var mutation struct {
	InsertParsedMessages struct {
		Returning []domain.ParsedMessage `json:"returning"`
	} `
	graphql:"insert_parsed_messages(objects: {
	startIndicator: $startIndicator,
	messageId: $messageId,
	dateTime: $dateTime,
	priorityIndicator: $priorityIndicator,
	primaryAddress: $primaryAddress,
	secondaryAddresses: $secondaryAddresses,
	originator: $originator,
	originatorDateTime: $originatorDateTime,
	category: $category,
	bodyAndFooter: $bodyAndFooter,
	bodyData: $bodyData,
	receivedAt: $receivedAt,
	parsedAt: $parsedAt,
	dispatchedAt: $dispatchedAt,
	needDispatch: $needDispatch
	})"`
}

// InsertParsedMessage inserts a new ParsedMessage into the Hasura GraphQL API
func (hr *HasuraRepository) InsertParsedMessage(pm domain.ParsedMessage) error {

	variables := map[string]interface{}{
		// "startIndicator":     graphql.String(pm.StartIndicator),
		"messageId":          graphql.String(pm.MessageID),
		"dateTime":           graphql.String(pm.DateTime),
		"priorityIndicator":  graphql.String(pm.PriorityIndicator),
		"primaryAddress":     graphql.String(pm.PrimaryAddress),
		"secondaryAddresses": pm.SecondaryAddresses,
		"originator":         graphql.String(pm.Originator),
		"originatorDateTime": graphql.String(pm.OriginatorDateTime),
		"category":           graphql.String(pm.Category),
		"bodyAndFooter":      graphql.String(pm.BodyAndFooter),
		"bodyData":           pm.BodyData,
		"receivedAt":         pm.ReceivedAt,
		"parsedAt":           pm.ParsedAt,
		"dispatchedAt":       pm.DispatchedAt,
		"needDispatch":       pm.NeedDispatch,
	}

	ctx := context.Background()
	err := hr.hasuraClient.Mutate(ctx, &mutation, variables)
	if err != nil {
		return err
	}

	for _, returnedMessage := range mutation.InsertParsedMessages.Returning {
		fmt.Printf("Inserted ParsedMessage: %+v\n", returnedMessage)
	}
	return nil
}
