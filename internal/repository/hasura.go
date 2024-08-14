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

type HasuraRepository struct {
	client graphql.Client
}

// New creates a new HasuraRepository
func NewHasura(config *config.Config) *HasuraRepository {
	token := os.Getenv("GRAPHQL_TOKEN")
	if token == "" {
		token = config.Hasura.Secret
	}
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	return &HasuraRepository{
		client: graphql.NewClient(config.Hasura.Endpoint, httpClient),
	}
}

// InsertParsedMessage inserts a new ParsedMessage into the Hasura GraphQL API
func (hr *HasuraRepository) CreateNew(pm *domain.ParsedMessage) error {
	log := utils.GetSugaredLogger()
	bodyString, _ := json.Marshal(pm.BodyData)
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
