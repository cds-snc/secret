package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Record struct {
	ID              string `dynamodbav:"id"`
	Data            []byte `dynamodbav:"data"`
	Key             []byte `dynamodbav:"key"`
	TTL             int64  `dynamodbav:"ttl"`
	ClaimToken      string `dynamodbav:"claim_token,omitempty"`
	ClaimUntil      int64  `dynamodbav:"claim_until,omitempty"`
	ClientEncrypted *bool  `dynamodbav:"client_encrypted,omitempty"`
}

type DynamoDBBackend struct {
	client     *dynamodb.Client
	table_name string
}

func (b *DynamoDBBackend) createTable() error {
	primary_key := "id"

	read_capacity := int64(5)
	write_capacity := int64(5)

	_, err := b.client.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: &primary_key,
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: &primary_key,
				KeyType:       types.KeyTypeHash,
			},
		},
		TableName: &b.table_name,
		ProvisionedThroughput: &types.ProvisionedThroughput{
			ReadCapacityUnits:  &read_capacity,
			WriteCapacityUnits: &write_capacity,
		},
	})

	return err
}

func (b *DynamoDBBackend) Delete(id uuid.UUID) error {
	_, err := b.client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id.String()},
		},
		TableName: &b.table_name,
	})

	return err
}

func (b *DynamoDBBackend) Init(c map[string]string) error {
	if c["region"] == "" {
		return fmt.Errorf("region is required")
	}

	if c["table_name"] == "" {
		return fmt.Errorf("table_name is required")
	}

	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(c["region"]))

	if err != nil {
		return err
	}

	b.client = dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if c["endpoint"] != "" {
			endpoint := c["endpoint"]
			o.BaseEndpoint = &endpoint
		}
	})
	b.table_name = c["table_name"]

	return nil
}

func (b *DynamoDBBackend) Store(data, key []byte, ttl int64, clientEncrypted bool) (uuid.UUID, error) {
	id := uuid.New()

	record := Record{
		ID:              id.String(),
		Data:            data,
		Key:             key,
		TTL:             ttl,
		ClientEncrypted: &clientEncrypted,
	}

	av, err := attributevalue.MarshalMap(record)

	if err != nil {
		return uuid.Nil, err
	}

	_, err = b.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		Item:      av,
		TableName: &b.table_name,
	})

	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

// Claim atomically reserves an unclaimed or abandoned secret and returns its data.
func (b *DynamoDBBackend) Claim(id uuid.UUID, lease time.Duration) (ClaimedSecret, error) {
	if lease <= 0 {
		return ClaimedSecret{}, fmt.Errorf("claim lease must be positive")
	}

	now := time.Now()
	token := uuid.New()
	updateExpression := "SET #claim_token = :claim_token, #claim_until = :claim_until"
	conditionExpression := "attribute_exists(#id) AND #ttl >= :now AND (attribute_not_exists(#claim_until) OR #claim_until <= :now)"
	record, err := b.client.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id.String()},
		},
		TableName:           &b.table_name,
		UpdateExpression:    &updateExpression,
		ConditionExpression: &conditionExpression,
		ExpressionAttributeNames: map[string]string{
			"#id":          "id",
			"#ttl":         "ttl",
			"#claim_token": "claim_token",
			"#claim_until": "claim_until",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":now":         &types.AttributeValueMemberN{Value: fmt.Sprint(now.Unix())},
			":claim_token": &types.AttributeValueMemberS{Value: token.String()},
			":claim_until": &types.AttributeValueMemberN{Value: fmt.Sprint(now.Add(lease).Unix())},
		},
		ReturnValues:                        types.ReturnValueAllNew,
		ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureAllOld,
	})

	if err != nil {
		if conditionErr := secretClaimConditionError(err); conditionErr != nil {
			return ClaimedSecret{}, conditionErr
		}
		return ClaimedSecret{}, err
	}

	if record.Attributes == nil {
		return ClaimedSecret{}, ErrSecretNotFound
	}

	var r Record
	err = attributevalue.UnmarshalMap(record.Attributes, &r)
	if err != nil {
		return ClaimedSecret{}, err
	}

	return ClaimedSecret{
		Data:            r.Data,
		Key:             r.Key,
		Token:           token,
		ClientEncrypted: r.ClientEncrypted,
	}, nil
}

// Consume permanently deletes a secret if the caller still owns an active claim.
func (b *DynamoDBBackend) Consume(id, token uuid.UUID) error {
	conditionExpression := "#claim_token = :claim_token AND #claim_until > :now"
	_, err := b.client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id.String()},
		},
		TableName:           &b.table_name,
		ConditionExpression: &conditionExpression,
		ExpressionAttributeNames: map[string]string{
			"#claim_token": "claim_token",
			"#claim_until": "claim_until",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":claim_token": &types.AttributeValueMemberS{Value: token.String()},
			":now":         &types.AttributeValueMemberN{Value: fmt.Sprint(time.Now().Unix())},
		},
	})

	if isConditionalCheckFailed(err) {
		return ErrClaimLost
	}
	return err
}

// Release clears a claim if the caller still owns it, allowing an immediate retry.
func (b *DynamoDBBackend) Release(id, token uuid.UUID) error {
	updateExpression := "REMOVE #claim_token, #claim_until"
	conditionExpression := "#claim_token = :claim_token"
	_, err := b.client.UpdateItem(context.TODO(), &dynamodb.UpdateItemInput{
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id.String()},
		},
		TableName:           &b.table_name,
		UpdateExpression:    &updateExpression,
		ConditionExpression: &conditionExpression,
		ExpressionAttributeNames: map[string]string{
			"#claim_token": "claim_token",
			"#claim_until": "claim_until",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":claim_token": &types.AttributeValueMemberS{Value: token.String()},
		},
	})

	if isConditionalCheckFailed(err) {
		return ErrClaimLost
	}
	return err
}

func isConditionalCheckFailed(err error) bool {
	var conditionFailed *types.ConditionalCheckFailedException
	return errors.As(err, &conditionFailed)
}

func secretClaimConditionError(err error) error {
	var conditionFailed *types.ConditionalCheckFailedException
	if !errors.As(err, &conditionFailed) {
		return nil
	}
	if len(conditionFailed.Item) == 0 {
		return ErrSecretNotFound
	}
	return ErrSecretUnavailable
}
