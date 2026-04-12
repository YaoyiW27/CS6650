package queue

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type PhotoMessage struct {
	PhotoID string `json:"photo_id"`
	AlbumID string `json:"album_id"`
}

type Queue struct {
	client   *sqs.Client
	queueURL string
}

func New(region, queueURL string) (*Queue, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return &Queue{
		client:   sqs.NewFromConfig(cfg),
		queueURL: queueURL,
	}, nil
}

func (q *Queue) Send(msg PhotoMessage) error {
	body, _ := json.Marshal(msg)
	_, err := q.client.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    &q.queueURL,
		MessageBody: aws.String(string(body)),
	})
	return err
}

func (q *Queue) Receive() (*PhotoMessage, *string, error) {
	out, err := q.client.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
		QueueUrl:            &q.queueURL,
		MaxNumberOfMessages: 1,
		WaitTimeSeconds:     10,
	})
	if err != nil || len(out.Messages) == 0 {
		return nil, nil, err
	}

	var msg PhotoMessage
	json.Unmarshal([]byte(*out.Messages[0].Body), &msg)
	return &msg, out.Messages[0].ReceiptHandle, nil
}

func (q *Queue) Delete(receiptHandle *string) error {
	_, err := q.client.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
		QueueUrl:      &q.queueURL,
		ReceiptHandle: receiptHandle,
	})
	return err
}