// main.go
package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	fiberadapter "github.com/awslabs/aws-lambda-go-api-proxy/fiber"
	app "github.com/cds-snc/secret"
	"github.com/cds-snc/secret/encryption"
	"github.com/cds-snc/secret/storage"
)

var fiberLambda *fiberadapter.FiberLambda

func init() {

	//Validate that all the required environment variables are set
	if os.Getenv("KMS_ID") == "" {
		panic("KMS_ID environment variable is not set")
	}

	if os.Getenv("DYNAMO_TABLE") == "" {
		panic("DYNAMO_TABLE environment variable is not set")
	}

	if os.Getenv("AWS_REGION") == "" {
		panic("AWS_REGION environment variable is not set")
	}

	encryption := &encryption.AwsKmsEncryption{}
	_ = encryption.Init(map[string]string{
		"kms_key_id": os.Getenv("KMS_ID"),
		"region":     os.Getenv("AWS_REGION"),
	})

	storage := &storage.DynamoDBBackend{}
	_ = storage.Init(map[string]string{
		"table_name": os.Getenv("DYNAMO_TABLE"),
		"region":     os.Getenv("AWS_REGION"),
	})

	a := app.CreateApp(encryption, storage)
	fiberLambda = fiberadapter.New(a)
}

// Handler will deal with Fiber working with Lambda
func Handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return fiberLambda.ProxyWithContextV2(ctx, req)
}

func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(Handler)
}
