package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"album-store/db"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PhotoHandler struct {
	DB       *db.DB
	S3Client *s3.Client
	Bucket   string
	Region   string
}

func (h *PhotoHandler) Upload(c *gin.Context) {
	albumID := c.Param("album_id")

	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing photo field"})
		return
	}
	defer file.Close()

	photoID := uuid.New().String()

	// stream to temp file instead of reading all into memory
	tmpFile, err := os.CreateTemp("", "photo-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp file"})
		return
	}
	tmpPath := tmpFile.Name()

	if _, err := io.Copy(tmpFile, file); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save upload"})
		return
	}
	tmpFile.Close()

	var seq int
	err = h.DB.QueryRow(`
		INSERT INTO album_seq (album_id, next_seq)
		VALUES ($1, 2)
		ON CONFLICT (album_id) DO UPDATE
		SET next_seq = album_seq.next_seq + 1
		RETURNING next_seq - 1
	`, albumID).Scan(&seq)
	if err != nil {
		os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign seq"})
		return
	}

	_, err = h.DB.Exec(`
		INSERT INTO photos (photo_id, album_id, seq, status)
		VALUES ($1, $2, $3, 'processing')
	`, photoID, albumID, seq)
	if err != nil {
		os.Remove(tmpPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create photo record"})
		return
	}

	// background: read temp file -> upload S3 -> cleanup
	go func() {
		defer os.Remove(tmpPath)

		f, err := os.Open(tmpPath)
		if err != nil {
			h.DB.Exec(`UPDATE photos SET status = 'failed' WHERE photo_id = $1`, photoID)
			return
		}
		defer f.Close()

		key := fmt.Sprintf("albums/%s/photos/%s", albumID, photoID)
		_, err = h.S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: &h.Bucket,
			Key:    &key,
			Body:   f,
		})
		if err != nil {
			h.DB.Exec(`UPDATE photos SET status = 'failed' WHERE photo_id = $1`, photoID)
			return
		}

		url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", h.Bucket, h.Region, key)
		h.DB.Exec(`UPDATE photos SET status = 'completed', url = $1 WHERE photo_id = $2`, url, photoID)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"photo_id": photoID,
		"seq":      seq,
		"status":   "processing",
	})
}

func (h *PhotoHandler) Get(c *gin.Context) {
	photoID := c.Param("photo_id")
	albumID := c.Param("album_id")

	var photo struct {
		PhotoID string  `json:"photo_id"`
		AlbumID string  `json:"album_id"`
		Seq     int     `json:"seq"`
		Status  string  `json:"status"`
		URL     *string `json:"url,omitempty"`
	}

	err := h.DB.QueryRow(`
		SELECT photo_id, album_id, seq, status, url
		FROM photos WHERE photo_id = $1 AND album_id = $2
	`, photoID, albumID).Scan(&photo.PhotoID, &photo.AlbumID, &photo.Seq, &photo.Status, &photo.URL)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, photo)
}

func (h *PhotoHandler) Delete(c *gin.Context) {
	photoID := c.Param("photo_id")
	albumID := c.Param("album_id")

	var url *string
	err := h.DB.QueryRow(
		`SELECT url FROM photos WHERE photo_id = $1 AND album_id = $2`,
		photoID, albumID,
	).Scan(&url)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	h.DB.Exec(`DELETE FROM photos WHERE photo_id = $1 AND album_id = $2`, photoID, albumID)

	if url != nil {
		key := fmt.Sprintf("albums/%s/photos/%s", albumID, photoID)
		h.S3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
			Bucket: &h.Bucket,
			Key:    &key,
		})
	}

	c.Status(http.StatusNoContent)
}