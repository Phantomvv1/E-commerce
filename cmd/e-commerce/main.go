package main

import (
	"net/http"

	. "github.com/Phantomvv1/E-commerce/internal/authentication"
	. "github.com/Phantomvv1/E-commerce/internal/cart"
	. "github.com/Phantomvv1/E-commerce/internal/items"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.Any("/", func(c *gin.Context) { c.JSON(http.StatusOK, nil) })
	r.POST("/signup", SignUp)
	r.POST("/login", LogIn)
	r.POST("/profile", GetCurrentProfile)
	r.POST("/profiles", GetAllUsers)
	r.POST("/item", CreateItem)
	r.PUT("/item", UpdateItem)
	r.POST("/item/get", GetItemByID)
	r.POST("/item/search", SearchForItem)
	r.GET("/items", GetAllItems)
	r.GET("/item/rand", GetRandomItem)
	r.DELETE("/item", DeleteItem)
	r.GET("/item/count", CountItems)
	r.POST("/cart/item", AddItemToCart)
	r.POST("/cart/items", GetItemsFromCart)

	r.Run(":42069")
}
