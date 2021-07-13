package tokens

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/common-fate/iamzero/pkg/crypto"
	"github.com/pkg/errors"
)

// DynamoDBTokenStorer is a token storage backend which uses DynamoDB
type DynamoDBTokenStorer struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoDBTokenStorer initialises the AWS DynamoDB client and returns a new DynamoDBTokenStorer
func NewDynamoDBTokenStorer(ctx context.Context, tableName string) (*DynamoDBTokenStorer, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}

	client := dynamodb.NewFromConfig(cfg)
	return &DynamoDBTokenStorer{client, tableName}, nil
}

// Create a Token and store it in the database
func (s *DynamoDBTokenStorer) Create(ctx context.Context, name string) (*Token, error) {
	ID, err := crypto.GenerateRandomToken()
	if err != nil {
		return nil, errors.Wrap(err, "generating token")
	}

	token := Token{
		ID:   ID,
		Name: name,
	}

	putItem, err := attributevalue.MarshalMap(token)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling item")
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: &s.tableName, Item: putItem})
	if err != nil {
		return nil, errors.Wrap(err, "putting item")
	}

	return &token, nil
}

// Delete a token from the database
func (s *DynamoDBTokenStorer) Delete(ctx context.Context, id string) error {

	input := &dynamodb.DeleteItemInput{
		TableName: &s.tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	}

	_, err := s.client.DeleteItem(ctx, input)
	return err
}

// Get a token from the database
func (s *DynamoDBTokenStorer) Get(ctx context.Context, id string) (*Token, error) {

	getItemInput := &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		TableName:      &s.tableName,
		ConsistentRead: aws.Bool(true),
	}

	getItemResponse, err := s.client.GetItem(ctx, getItemInput)

	if err != nil {
		return nil, err
	}

	if getItemResponse.Item == nil {
		return nil, ErrTokenNotFound
	}

	var token Token

	err = attributevalue.UnmarshalMap(getItemResponse.Item, &token)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalling item")
	}

	return &token, nil
}

// List all tokens
// TODO: currently this implementation uses DynamoDB scan
// To improve performance moving forwards to a production ready service
// we should paginate this and use Query instead.
func (s *DynamoDBTokenStorer) List(ctx context.Context) ([]Token, error) {
	input := &dynamodb.ScanInput{
		TableName: &s.tableName,
	}

	response, err := s.client.Scan(ctx, input)
	if err != nil {
		return nil, err
	}

	tokens := []Token{}
	if err := attributevalue.UnmarshalListOfMaps(response.Items, &tokens); err != nil {
		return nil, errors.Wrap(err, "unmarshalling items")
	}

	return tokens, nil

}
