package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Record struct {
	ID   string `dynamodbav:"id"`
	Data []byte `dynamodbav:"data"`
	Key  []byte `dynamodbav:"key"`
	TTL  int64  `dynamodbav:"ttl"`
}

type DynamoDBBackend struct {
	client     *dynamodb.Client
	table_name string
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

func (b *DynamoDBBackend) Store(data, key []byte, ttl int64) (uuid.UUID, error) {
	id := uuid.New()

	record := Record{
		ID:   id.String(),
		Data: data,
		Key:  key,
		TTL:  ttl,
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

func (b *DynamoDBBackend) Retrieve(id uuid.UUID) ([]byte, []byte, error) {
	record, err := b.client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id.String()},
		},
		TableName: &b.table_name,
	})

	if err != nil {
		return nil, nil, err
	}

	if record.Item == nil {
		return nil, nil, nil
	}

	var r Record

	err = attributevalue.UnmarshalMap(record.Item, &r)

	if err != nil {
		return nil, nil, err
	}

	// Check if TTL timestamp is less than current unix time
	if r.TTL < time.Now().Unix() {
		b.Delete(id)
		return nil, nil, fmt.Errorf("UUID not found")
	}

	return r.Data, r.Key, nil
}
