package handlers

import (
	"net/http"

	"album-store/db"
	"album-store/models"

	"github.com/gin-gonic/gin"
)

type AlbumHandler struct {
	DB *db.DB
}

func (h *AlbumHandler) Put(c *gin.Context) {
	albumID := c.Param("album_id")

	var req models.Album
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	req.AlbumID = albumID

	_, err := h.DB.Exec(`
		INSERT INTO albums (album_id, title, description, owner)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (album_id) DO UPDATE
		SET title = $2, description = $3, owner = $4
	`, req.AlbumID, req.Title, req.Description, req.Owner)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// xmax == 0 means newly inserted (201), otherwise updated (200)
	var xmax int
	h.DB.QueryRow(`SELECT xmax FROM albums WHERE album_id = $1`, req.AlbumID).Scan(&xmax)
	status := http.StatusOK
	if xmax == 0 {
		status = http.StatusCreated
	}

	c.JSON(status, req)
}

func (h *AlbumHandler) Get(c *gin.Context) {
	albumID := c.Param("album_id")

	var album models.Album
	err := h.DB.QueryRow(`
		SELECT album_id, title, description, owner
		FROM albums WHERE album_id = $1
	`, albumID).Scan(&album.AlbumID, &album.Title, &album.Description, &album.Owner)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, album)
}

func (h *AlbumHandler) List(c *gin.Context) {
	rows, err := h.DB.Query(`SELECT album_id, title, description, owner FROM albums`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	albums := []models.Album{}
	for rows.Next() {
		var a models.Album
		if err := rows.Scan(&a.AlbumID, &a.Title, &a.Description, &a.Owner); err != nil {
			continue
		}
		albums = append(albums, a)
	}

	c.JSON(http.StatusOK, albums)
}