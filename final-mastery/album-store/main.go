package main

import (
	"context"
	"log"

	"album-store/config"
	"album-store/db"
	"album-store/handlers"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatalf("failed to migrate: %v", err)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(), awsconfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		log.Fatalf("failed to load aws config: %v", err)
	}
	s3Client := s3.NewFromConfig(awsCfg)

	albumHandler := &handlers.AlbumHandler{DB: database}
	photoHandler := &handlers.PhotoHandler{
		DB:       database,
		S3Client: s3Client,
		Bucket:   cfg.S3Bucket,
		Region:   cfg.AWSRegion,
	}

	r := gin.Default()

	r.GET("/health", handlers.Health)
	r.PUT("/albums/:album_id", albumHandler.Put)
	r.GET("/albums/:album_id", albumHandler.Get)
	r.GET("/albums", albumHandler.List)

	r.POST("/albums/:album_id/photos", photoHandler.Upload)
	r.GET("/albums/:album_id/photos/:photo_id", photoHandler.Get)
	r.DELETE("/albums/:album_id/photos/:photo_id", photoHandler.Delete)

	log.Printf("starting server on :%s", cfg.Port)
	r.Run(":" + cfg.Port)
}