package cart

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	. "github.com/Phantomvv1/E-commerce/internal/items"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Cart struct {
	Items []Item `json:"items"`
}

func CreateCartTable(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), "create table if not exists e_commerce.cart (id serial primary key, item_id int references e_commerce.items (id)"+
		", user_id int references e_commerce.authentication(id))")
	return err
}

func AddItemToCart(c *gin.Context) {
	var information map[string]int
	json.NewDecoder(c.Request.Body).Decode(&information) // userID && itemID

	userID, ok := information["userID"]
	if !ok {
		log.Println("Incorrectly provided id of the user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided id of the user"})
		return
	}

	itemID, ok := information["itemID"]
	if !ok {
		log.Println("Incorrectly provided id of the item")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided id of the item"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	if err = CreateCartTable(conn); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to create a table for the cart"})
		return
	}

	_, err = conn.Exec(context.Background(), "insert into e_commerce.cart (item_id, user_id) values ($1, $2)", itemID, userID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to put the information in the database"})
		return
	}

	c.JSON(http.StatusOK, nil)
}
