package store

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBStore implements PackSizeStore using a single DynamoDB item.
type DynamoDBStore struct {
	client    *dynamodb.Client
	tableName string
}

const packSizesPK = "PACK_SIZES"

type packSizesData struct {
	PK      string `dynamodbav:"pk"`
	Sizes   []int  `dynamodbav:"sizes"`
	Version int    `dynamodbav:"version"`
}

// NewDynamoDBStore creates a DynamoDB-backed store and seeds it with
// defaultSizes if no item exists yet.
func NewDynamoDBStore(client *dynamodb.Client, tableName string, defaultSizes []int) (*DynamoDBStore, error) {
	s := &DynamoDBStore{client: client, tableName: tableName}

	sizes, err := s.GetAll()
	if err != nil {
		return nil, err
	}
	if len(sizes) == 0 {
		if err := s.initDefaults(defaultSizes); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *DynamoDBStore) GetAll() ([]int, error) {
	data, err := s.readData()
	if err != nil {
		return nil, err
	}
	sort.Ints(data.Sizes)
	return data.Sizes, nil
}

func (s *DynamoDBStore) Add(size int) error {
	return s.modifySizes(func(sizes []int) ([]int, error) {
		for _, existing := range sizes {
			if existing == size {
				return nil, ErrPackSizeExists
			}
		}
		return append(sizes, size), nil
	})
}

func (s *DynamoDBStore) Remove(size int) error {
	return s.modifySizes(func(sizes []int) ([]int, error) {
		if len(sizes) <= 1 {
			return nil, ErrLastPackSize
		}
		out := make([]int, 0, len(sizes)-1)
		found := false
		for _, v := range sizes {
			if v == size {
				found = true
			} else {
				out = append(out, v)
			}
		}
		if !found {
			return nil, ErrPackSizeNotFound
		}
		return out, nil
	})
}

// modifySizes reads the current sizes, applies the provided modification function,
// and writes the updated sizes back to DynamoDB, ensuring that concurrent modifications
// are detected and retried.
func (s *DynamoDBStore) modifySizes(fn func([]int) ([]int, error)) error {
	write := func() error {
		data, err := s.readData()
		if err != nil {
			return err
		}

		newSizes, err := fn(data.Sizes)
		if err != nil {
			return err
		}

		return s.writeData(packSizesData{
			PK:      packSizesPK,
			Sizes:   newSizes,
			Version: data.Version + 1,
		}, data.Version)

	}

	cfg := retryConfig{
		MaxRetries:   3,
		BackoffDelay: 50 * time.Millisecond,
		ShouldRetry: func(err error) bool {
			var condErr *types.ConditionalCheckFailedException
			return errors.As(err, &condErr)
		},
	}

	return retry(write, cfg)
}

func (s *DynamoDBStore) readData() (packSizesData, error) {
	result, err := s.client.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: packSizesPK},
		},
	})
	if err != nil {
		return packSizesData{}, err
	}
	if len(result.Item) == 0 {
		return packSizesData{PK: packSizesPK}, nil
	}
	var data packSizesData
	if err := attributevalue.UnmarshalMap(result.Item, &data); err != nil {
		return packSizesData{}, err
	}
	return data, nil
}

// writeData persists data only when the stored version matches expectedVersion
// (or the item does not yet exist).
func (s *DynamoDBStore) writeData(data packSizesData, expectedVersion int) error {
	item, err := attributevalue.MarshalMap(data)
	if err != nil {
		return err
	}
	_, err = s.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(pk) OR #v = :expected"),
		ExpressionAttributeNames: map[string]string{
			"#v": "version",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":expected": &types.AttributeValueMemberN{Value: strconv.Itoa(expectedVersion)},
		},
	})
	return err
}

// initDefaults writes the default sizes only if the item does not exist yet.
func (s *DynamoDBStore) initDefaults(defaultSizes []int) error {
	data := packSizesData{PK: packSizesPK, Sizes: defaultSizes, Version: 0}
	item, err := attributevalue.MarshalMap(data)
	if err != nil {
		return err
	}
	_, err = s.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(pk)"),
	})
	var condErr *types.ConditionalCheckFailedException
	if errors.As(err, &condErr) {
		return nil // another instance already seeded the item
	}
	return err
}

type retryConfig struct {
	MaxRetries   int
	ShouldRetry  func(error) bool
	BackoffDelay time.Duration // Exponential delay
}

func retry(fn func() error, cfg retryConfig) error {
	for attempt := 0; attempt < cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt)) * cfg.BackoffDelay
			time.Sleep(delay)
		}

		err := fn()
		if err == nil {
			return nil
		}

		if cfg.ShouldRetry == nil || !cfg.ShouldRetry(err) {
			return err
		}
	}

	return errors.New("concurrent modification detected, please retry")
}
