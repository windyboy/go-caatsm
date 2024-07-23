package repository

import (
	"context"
	"encoding/json"
	"os"

	"caatsm/internal/domain"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

type HasuraRepository struct {
	client graphql.Client
}

// NewHasuraRepo creates a new HasuraRepository
func NewHasuraRepo(endpoint, secret string) *HasuraRepository {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GRAPHQL_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	return &HasuraRepository{
		client: graphql.NewClient(endpoint, httpClient),
	}
}

// InsertParsedMessage inserts a new ParsedMessage into the Hasura GraphQL API
func (hr *HasuraRepository) InsertParsedMessage(pm *domain.ParsedMessage) error {
	bodyString, _ := json.Marshal(pm.BodyData)
	variables := Aviation_telegrams_insert_input{
		// Id:              10,
		Body_and_footer: pm.BodyAndFooter,
		Body_data:       string(bodyString),
		Category:        pm.Category,
		Date_time:       pm.DateTime,
		Dispatched_at:   pm.DispatchedAt,
		Uuid:            uuid.New(),
	}
	_, err := newMessage(context.Background(), hr.client, variables)
	if err != nil {
		return err
	}
	// fmt.Printf("Inserted new message with ID: %v\n", resp)
	return nil
}
