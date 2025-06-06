package comparison

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

type Comparison struct {
	Items []Item `json:"items"`
}

func CreateComparisonTable(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), "create table if not exists e_commerce.comparison (user_id int references e_commerce.authentication(id) on delete cascade, "+
		"item_id int references e_commerce.items(id) on delete cascade)")
	return err
}

func ItemExists(conn *pgx.Conn, itemID int) (bool, error) {
	id := 0
	err := conn.QueryRow(context.Background(), "select id from e_commerce.items where id = $1", itemID).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func AlreadyCompareingItem(conn *pgx.Conn, itemID, userID int) (bool, error) {
	check := 0
	err := conn.QueryRow(context.Background(), "select item_id from e_commerce.comparison where user_id = $1 and item_id = $2", userID, itemID).Scan(&check)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func AddItemToCompare(c *gin.Context) {
	var information map[string]interface{}
	json.NewDecoder(c.Request.Body).Decode(&information) // token && itemID

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

	itemIDFl, ok := information["itemID"].(float64)
	if !ok {
		log.Println("Incorrectly provided the id of the user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided the id of the user"})
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

	if err = CreateComparisonTable(conn); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to create a table for the comparison"})
		return
	}

	exists, err := ItemExists(conn, itemID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking if an item with such an ID exists"})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Error there is no item with this id in this shop"})
		return
	}

	compareing, err := AlreadyCompareingItem(conn, itemID, id)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error couldn't get information from the database if this item is already in the user's comparison list"})
		return
	}

	if compareing {
		c.JSON(http.StatusConflict, gin.H{"error": "Error trying to add an item to the comaprison that is already in the comparison"})
		return
	}

	_, err = conn.Exec(context.Background(), "insert into e_commerce.comparison (user_id, item_id) values ($1, $2)", id, itemID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to put the information in the database"})
		return
	}

	c.JSON(http.StatusOK, nil)
}

func Compare(c *gin.Context) {
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

	rows, err := conn.Query(context.Background(), "select item_id from e_commerce.comparison where user_id = $1 order by item_id", id)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get the information about the items you want to compare from the database"})
		return
	}

	var itemIDs []interface{}
	for rows.Next() {
		itemID := 0
		err = rows.Scan(&itemID)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get the id-s of the items you want to compare"})
			return
		}

		itemIDs = append(itemIDs, itemID)
	}

	if rows.Err() != nil {
		log.Println(rows.Err())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting the id-s of the items"})
		return
	}

	query := "select name, description, price from e_commerce.items where id in ("
	for i := range len(itemIDs) {
		if i == len(itemIDs)-1 {
			query += "$" + fmt.Sprintf("%d", i+1)
		} else {
			query += "$" + fmt.Sprintf("%d", i+1) + ", "
		}
	}
	query += ") order by id"

	if query == "select name, description, price from e_commerce.items where id in () order by id" {
		c.JSON(http.StatusOK, gin.H{"message": "There are no items in this user's comparison"})
		return
	}

	rows, err = conn.Query(context.Background(), query, itemIDs...)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get the items you want to compare from the database"})
		return
	}

	items := []Item{}
	i := 0
	for rows.Next() {
		item := Item{}
		err = rows.Scan(&item.Name, &item.Description, &item.Price)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to correctly get all the information about the items from the database"})
			return
		}

		item.ID = itemIDs[i].(int)
		i++

		items = append(items, item)
	}

	if rows.Err() != nil {
		log.Println(rows.Err())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to work with the items you have selected"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func RemoveItemFromComparison(c *gin.Context) {
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

	check := 0
	err = conn.QueryRow(context.Background(), "delete from e_commerce.comparison where user_id = $1 and item_id = $2 returning user_id", id, itemID).Scan(&check)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Error there is no item with this id in your comparison list"})
			return
		}

		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to remove this item from your comparison list"})
		return
	}

	c.JSON(http.StatusOK, nil)
}

func RemoveAllItemsFromComparison(c *gin.Context) {
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

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(context.Background(), "delete from e_commerce.comparison where user_id = $1", id)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to remove all of the items in this user's comparison"})
		return
	}

	c.JSON(http.StatusOK, nil)
}
