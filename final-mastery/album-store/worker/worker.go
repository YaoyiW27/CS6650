package worker

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"album-store/db"
	"album-store/queue"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Worker struct {
	DB       *db.DB
	Queue    *queue.Queue
	S3Client *s3.Client
	Bucket   string
	Region   string
}

func New(database *db.DB, q *queue.Queue, region, bucket string) (*Worker, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	return &Worker{
		DB:       database,
		Queue:    q,
		S3Client: s3.NewFromConfig(cfg),
		Bucket:   bucket,
		Region:   region,
	}, nil
}

func (w *Worker) Start() {
	go func() {
		for {
			msg, receipt, err := w.Queue.Receive()
			if err != nil {
				log.Printf("queue receive error: %v", err)
				continue
			}
			if msg == nil {
				continue
			}
			w.process(msg)
			w.Queue.Delete(receipt)
		}
	}()
}

func (w *Worker) process(msg *queue.PhotoMessage) {
	// read photo data from db temp storage
	var data []byte
	err := w.DB.QueryRow(`SELECT data FROM photo_data WHERE photo_id = $1`, msg.PhotoID).Scan(&data)
	if err != nil {
		log.Printf("failed to read photo data: %v", err)
		w.DB.Exec(`UPDATE photos SET status = 'failed' WHERE photo_id = $1`, msg.PhotoID)
		return
	}

	// upload to S3
	key := fmt.Sprintf("albums/%s/photos/%s", msg.AlbumID, msg.PhotoID)
	_, err = w.S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &w.Bucket,
		Key:    &key,
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		log.Printf("failed to upload to S3: %v", err)
		w.DB.Exec(`UPDATE photos SET status = 'failed' WHERE photo_id = $1`, msg.PhotoID)
		return
	}

	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", w.Bucket, w.Region, key)
	w.DB.Exec(`UPDATE photos SET status = 'completed', url = $1 WHERE photo_id = $2`, url, msg.PhotoID)

	// clean up temp data
	w.DB.Exec(`DELETE FROM photo_data WHERE photo_id = $1`, msg.PhotoID)
}