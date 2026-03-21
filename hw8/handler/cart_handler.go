package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"hw8/store"
)

type CartHandler struct {
	store store.CartStore
}

func NewCartHandler(s store.CartStore) *CartHandler {
	return &CartHandler{store: s}
}

// POST /shopping-carts
func (h *CartHandler) CreateCart(c *gin.Context) {
	var req struct {
		CustomerID int `json:"customer_id" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_INPUT",
			"message": err.Error(),
		})
		return
	}

	id, err := h.store.CreateCart(c.Request.Context(), req.CustomerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL_ERROR",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"shopping_cart_id": id,
	})
}

// GET /shopping-carts/:id
func (h *CartHandler) GetCart(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_INPUT",
			"message": "invalid cart id",
		})
		return
	}

	cart, err := h.store.GetCart(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL_ERROR",
			"message": err.Error(),
		})
		return
	}
	if cart == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "NOT_FOUND",
			"message": "cart not found",
		})
		return
	}

	c.JSON(http.StatusOK, cart)
}

// POST /shopping-carts/:id/items
func (h *CartHandler) AddItem(c *gin.Context) {
	cartID, err := strconv.Atoi(c.Param("id"))
	if err != nil || cartID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_INPUT",
			"message": "invalid cart id",
		})
		return
	}

	var req struct {
		ProductID int `json:"product_id" binding:"required,min=1"`
		Quantity  int `json:"quantity" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_INPUT",
			"message": err.Error(),
		})
		return
	}

	err = h.store.AddItem(c.Request.Context(), cartID, req.ProductID, req.Quantity)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "cart "+strconv.Itoa(cartID)+" not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "NOT_FOUND",
				"message": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL_ERROR",
			"message": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// RegisterRoutes wires up all cart endpoints.
func (h *CartHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/shopping-carts", h.CreateCart)
	r.GET("/shopping-carts/:id", h.GetCart)
	r.POST("/shopping-carts/:id/items", h.AddItem)
}