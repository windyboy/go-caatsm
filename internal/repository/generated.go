// Code generated by github.com/Khan/genqlient, DO NOT EDIT.

package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/google/uuid"
)

// input type for inserting data into table "aviation.telegrams"
type Aviation_telegrams_insert_input struct {
	Text                 string          `json:"Text"`
	Body_data            json.RawMessage `json:"body_data"`
	Category             string          `json:"category"`
	Date_time            string          `json:"date_time"`
	Dispatched_at        time.Time       `json:"dispatched_at"`
	Message_id           string          `json:"message_id"`
	Need_dispatch        bool            `json:"need_dispatch"`
	Originator           string          `json:"originator"`
	Originator_date_time string          `json:"originator_date_time"`
	Parsed_at            time.Time       `json:"parsed_at"`
	Primary_address      string          `json:"primary_address"`
	Priority_indicator   string          `json:"priority_indicator"`
	Received_at          time.Time       `json:"received_at"`
	Secondary_addresses  string          `json:"secondary_addresses"`
	Uuid                 uuid.UUID       `json:"uuid"`
}

// GetText returns Aviation_telegrams_insert_input.Text, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetText() string { return v.Text }

// GetBody_data returns Aviation_telegrams_insert_input.Body_data, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetBody_data() json.RawMessage { return v.Body_data }

// GetCategory returns Aviation_telegrams_insert_input.Category, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetCategory() string { return v.Category }

// GetDate_time returns Aviation_telegrams_insert_input.Date_time, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetDate_time() string { return v.Date_time }

// GetDispatched_at returns Aviation_telegrams_insert_input.Dispatched_at, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetDispatched_at() time.Time { return v.Dispatched_at }

// GetMessage_id returns Aviation_telegrams_insert_input.Message_id, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetMessage_id() string { return v.Message_id }

// GetNeed_dispatch returns Aviation_telegrams_insert_input.Need_dispatch, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetNeed_dispatch() bool { return v.Need_dispatch }

// GetOriginator returns Aviation_telegrams_insert_input.Originator, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetOriginator() string { return v.Originator }

// GetOriginator_date_time returns Aviation_telegrams_insert_input.Originator_date_time, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetOriginator_date_time() string {
	return v.Originator_date_time
}

// GetParsed_at returns Aviation_telegrams_insert_input.Parsed_at, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetParsed_at() time.Time { return v.Parsed_at }

// GetPrimary_address returns Aviation_telegrams_insert_input.Primary_address, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetPrimary_address() string { return v.Primary_address }

// GetPriority_indicator returns Aviation_telegrams_insert_input.Priority_indicator, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetPriority_indicator() string { return v.Priority_indicator }

// GetReceived_at returns Aviation_telegrams_insert_input.Received_at, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetReceived_at() time.Time { return v.Received_at }

// GetSecondary_addresses returns Aviation_telegrams_insert_input.Secondary_addresses, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetSecondary_addresses() string {
	return v.Secondary_addresses
}

// GetUuid returns Aviation_telegrams_insert_input.Uuid, and is useful for accessing the field via an interface.
func (v *Aviation_telegrams_insert_input) GetUuid() uuid.UUID { return v.Uuid }

// __newMessageInput is used internally by genqlient
type __newMessageInput struct {
	Object Aviation_telegrams_insert_input `json:"object"`
}

// GetObject returns __newMessageInput.Object, and is useful for accessing the field via an interface.
func (v *__newMessageInput) GetObject() Aviation_telegrams_insert_input { return v.Object }

// newMessageInsert_aviation_telegrams_oneAviation_telegrams includes the requested fields of the GraphQL type aviation_telegrams.
// The GraphQL type's documentation follows.
//
// columns and relationships of "aviation.telegrams"
type newMessageInsert_aviation_telegrams_oneAviation_telegrams struct {
	Message_id string    `json:"message_id"`
	Uuid       uuid.UUID `json:"uuid"`
}

// GetMessage_id returns newMessageInsert_aviation_telegrams_oneAviation_telegrams.Message_id, and is useful for accessing the field via an interface.
func (v *newMessageInsert_aviation_telegrams_oneAviation_telegrams) GetMessage_id() string {
	return v.Message_id
}

// GetUuid returns newMessageInsert_aviation_telegrams_oneAviation_telegrams.Uuid, and is useful for accessing the field via an interface.
func (v *newMessageInsert_aviation_telegrams_oneAviation_telegrams) GetUuid() uuid.UUID {
	return v.Uuid
}

// newMessageResponse is returned by newMessage on success.
type newMessageResponse struct {
	// insert a single row into the table: "aviation.telegrams"
	Insert_aviation_telegrams_one newMessageInsert_aviation_telegrams_oneAviation_telegrams `json:"insert_aviation_telegrams_one"`
}

// GetInsert_aviation_telegrams_one returns newMessageResponse.Insert_aviation_telegrams_one, and is useful for accessing the field via an interface.
func (v *newMessageResponse) GetInsert_aviation_telegrams_one() newMessageInsert_aviation_telegrams_oneAviation_telegrams {
	return v.Insert_aviation_telegrams_one
}

// The query or mutation executed by newMessage.
const newMessage_Operation = `
mutation newMessage ($object: aviation_telegrams_insert_input!) {
	insert_aviation_telegrams_one(object: $object) {
		message_id
		uuid
	}
}
`

func newMessage(
	ctx_ context.Context,
	client_ graphql.Client,
	object Aviation_telegrams_insert_input,
) (*newMessageResponse, error) {
	req_ := &graphql.Request{
		OpName: "newMessage",
		Query:  newMessage_Operation,
		Variables: &__newMessageInput{
			Object: object,
		},
	}
	var err_ error

	var data_ newMessageResponse
	resp_ := &graphql.Response{Data: &data_}

	err_ = client_.MakeRequest(
		ctx_,
		req_,
		resp_,
	)

	return &data_, err_
}
