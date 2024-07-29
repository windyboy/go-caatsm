package repository

import (
	"context"
	"encoding/json"
	"os"

	"caatsm/internal/domain"
	"caatsm/pkg/utils"

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
func (hr *HasuraRepository) CreateNew(pm *domain.ParsedMessage) error {
	log := utils.GetSugaredLogger()
	bodyString, _ := json.Marshal(pm.BodyData)
	secondAddress, _ := json.Marshal(pm.SecondaryAddresses)
	variables := Aviation_telegrams_insert_input{
		Message_id:           pm.MessageID,
		Priority_indicator:   pm.PriorityIndicator,
		Primary_address:      pm.PrimaryAddress,
		Secondary_addresses:  []byte(secondAddress),
		Text:                 pm.Text,
		Body_data:            bodyString,
		Category:             pm.Category,
		Date_time:            pm.DateTime,
		Dispatched_at:        pm.DispatchedAt,
		Uuid:                 uuid.New(),
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
