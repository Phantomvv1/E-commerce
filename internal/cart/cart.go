package cart

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	. "github.com/Phantomvv1/E-commerce/internal/authentication"
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

func GetItemsFromCart(c *gin.Context) {
	var information map[string]string
	json.NewDecoder(c.Request.Body).Decode(&information)

	token, ok := information["token"]
	if !ok {
		log.Println("Incorrectly provided token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided token"})
		return
	}

	id, _, err := ValidateJWT(token)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error invalid token"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), "select item_id from e_commerce.cart where user_id = $1", id)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get information from the database"})
		return
	}

	var itemIDs []interface{}
	for rows.Next() {
		itemID := 0
		err = rows.Scan(&itemID)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error working with the information from the database"})
			return
		}

		itemIDs = append(itemIDs, itemID)
	}

	if rows.Err() != nil {
		log.Println(rows.Err())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error working with the information from the database"})
		return
	}

	query := "select name, description, price from e_commerce.items where id in ("

	for i := range itemIDs {
		query += "$" + fmt.Sprintf("%d", i+1)
	}

	query += ")"

	rows, err = conn.Query(context.Background(), query, itemIDs...)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get information about the items"})
		return
	}

	var items []Item
	index := 0
	for rows.Next() {
		item := Item{}
		err = rows.Scan(&item.Name, &item.Description, &item.Price)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error working with the information about the items"})
			return
		}

		item.ID, ok = itemIDs[index].(int)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "What? This should have never happened!"})
			return
		}

		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{"cart": items})
}
