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

	_, err = conn.Exec(context.Background(), "insert into table e_commerce.comparison (user_id, item_id) values ($1, $2)", id, itemID)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to coonect to the database"})
		return
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), "select itemID from e_commerce.comparison where userID = $1", id)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get the information about the items you want to compare from the database"})
		return
	}

	itemIDs := []int{}
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
}
