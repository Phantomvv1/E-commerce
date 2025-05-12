package cart

import (
	"context"
	"encoding/json"
	"errors"
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

func getCartPrice(conn *pgx.Conn, userId int) (float32, error) {
	rows, err := conn.Query(context.Background(), "select c.item_id from e_commerce.cart c where user_id = $1", userId)
	if err != nil {
		return 0.0, err
	}

	var itemIDs []interface{}
	for rows.Next() {
		itemID := 0
		err = rows.Scan(&itemID)
		if err != nil {
			return 0.0, err
		}

		itemIDs = append(itemIDs, itemID)
	}

	query := "select sum(i.price) from e_commerce.items i where i.id in ("
	for i := range itemIDs {
		if i == len(itemIDs)-1 {
			query += "$" + fmt.Sprintf("%d", i+1)
		} else {
			query += "$" + fmt.Sprintf("%d", i+1) + ", "
		}
	}
	query += ")"

	if len(itemIDs) == 0 {
		return 0.0, errors.New("There are no items in your cart")
	}

	var price float32
	err = conn.QueryRow(context.Background(), query, itemIDs...).Scan(&price)
	if err != nil {
		return 0.0, err
	}

	return price, nil
}

func CreateCartTable(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), "create table if not exists e_commerce.cart (id serial primary key, item_id int references e_commerce.items (id)"+
		", user_id int references e_commerce.authentication(id))")
	return err
}

func AddItemToCart(c *gin.Context) {
	var information map[string]interface{}
	json.NewDecoder(c.Request.Body).Decode(&information) // token && itemID

	token, ok := information["token"].(string)
	if !ok {
		log.Println("Incorrectly provided token of the user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided token of the user"})
		return
	}

	userID, _, err := ValidateJWT(token)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error invalid token"})
		return
	}

	itemIDFl, ok := information["itemID"].(float64)
	if !ok {
		log.Println("Incorrectly provided id of the item")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided id of the item"})
		return
	}
	itemID := int(itemIDFl)

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

	rows, err := conn.Query(context.Background(), "select item_id from e_commerce.cart where user_id = $1 order by item_id", id)
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
		if i == len(itemIDs)-1 {
			query += "$" + fmt.Sprintf("%d", i+1)
		} else {
			query += "$" + fmt.Sprintf("%d", i+1) + ", "
		}
	}

	query += ") order by id"

	if query == "select name, description, price from e_commerce.items where id in () order by id" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Error there are no items in your cart"})
		return
	}

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

		item.ID, _ = itemIDs[index].(int)
		index++
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{"cart": items})
}

func RemoveItemFromCart(c *gin.Context) {
	var information map[string]interface{}
	json.NewDecoder(c.Request.Body).Decode(&information)

	token, ok := information["token"].(string)
	if !ok {
		log.Println("Incorrectly provided token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided token"})
		return
	}

	id, _, err := ValidateJWT(token)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error invalid token"})
		return
	}

	itemID, ok := information["itemID"].(float64)
	if !ok {
		log.Println("Id of the item is not provided correctly")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error id of the item is not provided correctly"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), "delete from e_commerce.cart where user_id = $1 and item_id = $2", id, itemID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to remove the item from your cart"})
		return
	}

	c.JSON(http.StatusOK, nil)
}

func CountItemsInCart(c *gin.Context) { // to test
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error invalid token"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	count := 0
	err = conn.QueryRow(context.Background(), "select count(*) from e_commerce.cart c where c.id = $1", id).Scan(&count)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error couldn't get information from the database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

func Checkout(c *gin.Context) {
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error invalid token"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	price, err := getCartPrice(conn, id)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get information from the database"})
		return
	}

	// TODO: Add real payment later
	log.Println(price)

	_, err = conn.Exec(context.Background(), "delete from e_commerce.cart where user_id = $1", id)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to remove the items from the cart after paying"})
		return
	}

	c.JSON(http.StatusOK, nil)
}

func RemoveEverythingFromCart(c *gin.Context) {
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error invalid token"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	check := 0
	err = conn.QueryRow(context.Background(), "delete from e_commerce.cart where user_id = $1 returning id", id).Scan(&check)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Error there are no items in your cart"})
			return
		}

		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to remove the items from your cart"})
		return
	}

	c.JSON(http.StatusOK, nil)
}

func GetCartPrice(c *gin.Context) {
	var information map[string]string
	json.NewDecoder(c.Request.Body).Decode(&information)

	token, ok := information["token"]
	if !ok {
		log.Println("Incorrectly provided token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Incorrectly provided token"})
		return
	}

	id, _, err := ValidateJWT(token)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error invalid token"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	price, err := getCartPrice(conn, id)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"price": price})
}
