package main

import (
	"net/http"

	. "github.com/Phantomvv1/E-commerce/internal/authentication"
	. "github.com/Phantomvv1/E-commerce/internal/cart"
	. "github.com/Phantomvv1/E-commerce/internal/comparison"
	. "github.com/Phantomvv1/E-commerce/internal/emails"
	. "github.com/Phantomvv1/E-commerce/internal/items"
	. "github.com/Phantomvv1/E-commerce/internal/wishlist"
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
	r.DELETE("/cart/item", RemoveItemFromCart)
	r.POST("/cart/item/count", CountItemsInCart)
	r.POST("/cart/pay", Checkout)
	r.DELETE("/cart/all", RemoveEverythingFromCart)
	r.POST("/cart/price", GetCartPrice)
	r.POST("/wishlist", PutItemInWishlist)
	r.POST("/wishlist/item", GetItemFromWishlist)
	r.POST("/wishlist/items", GetAllItemsFromWishlist)
	r.DELETE("/wishlist/item", RemoveItemFromWishlist)
	r.POST("/coupon", ApplyCoupon)
	r.DELETE("/coupon", RemoveCoupon)
	r.POST("/compare/item", AddItemToCompare)
	r.POST("/compare", Compare)
	r.DELETE("/compare/item", RemoveItemFromComparison)
	r.DELETE("/compare/items", RemoveAllItemsFromComparison)
	r.POST("email", SendEmail)

	r.Run(":42069")
}
