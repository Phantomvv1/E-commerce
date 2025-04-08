package items

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	. "github.com/Phantomvv1/E-commerce/internal/authentication"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Item struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func CreateItemsTable(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), "create table if not exists e_commerce.items (id serial primary key, name text, description text)")
	return err
}

func CreateItem(c *gin.Context) {
	var information map[string]interface{}
	json.NewDecoder(c.Request.Body).Decode(&information) // token && name && description

	token, ok := information["token"].(string)
	if !ok {
		log.Println("Token provided incorrectly")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error token provided incorrectly"})
		return
	}

	_, accountType, err := ValidateJWT(token)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error invalid token"})
		return
	}

	if accountType != Admin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Error only admins can create items"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	if err = CreateItemsTable(conn); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to create a table for the items"})
		return
	}

	name, ok := information["name"].(string)
	if !ok {
		log.Println("Incorrectly provided name of the item")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided name of the item"})
		return
	}

	desc, ok := information["description"].(string)
	if !ok {
		log.Println("Incorrectly provided description of the item")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided description of the item"})
		return
	}

	_, err = conn.Exec(context.Background(), "insert into e_commerce.items (name, description) values ($1, $2)", name, desc)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to put the information about the item in the database"})
		return
	}

	c.JSON(http.StatusOK, nil)
}

func UpdateItem(c *gin.Context) {
	var information map[string]interface{}
	json.NewDecoder(c.Request.Body).Decode(&information) // token && id && (name || desc)

	token, ok := information["token"].(string)
	if !ok {
		log.Println("Incorrectly provided token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided token"})
		return
	}

	_, accountType, err := ValidateJWT(token)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error invalid token"})
		return
	}

	if accountType != Admin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Error only admins can update items"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	id, ok := information["id"].(float64)
	if !ok {
		log.Println("Incorrectly provided id")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided id"})
		return
	}
	itemID := int(id)

	updateName := true
	name, ok := information["name"].(string)
	if !ok {
		updateName = false
	}

	updateDesc := true
	desc, ok := information["description"].(string)
	if !ok && !updateName {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error no information provided for the update"})
		return
	} else if !ok {
		updateDesc = false
	}

	if updateName && updateDesc {
		_, err = conn.Exec(context.Background(), "update e_commerce.items set name = $1, description = $2 where id = $3", name, desc, itemID)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to update the information in the database"})
			return
		}
	} else if updateName {
		_, err = conn.Exec(context.Background(), "update e_commerce.items set name = $1 where id = $2", name, itemID)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to update the information in the database"})
			return
		}
	} else {
		fmt.Println("Here")
		_, err = conn.Exec(context.Background(), "update e_commerce.items set description = $1 where id = $2", desc, itemID)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to update the information in the database"})
			return
		}
	}

	c.JSON(http.StatusOK, nil)
}

func GetItemByID(c *gin.Context) {
	var information map[string]int
	json.NewDecoder(c.Request.Body).Decode(&information)

	id, ok := information["id"]
	if !ok {
		log.Println("Incorrectly provided id")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided id"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	item := Item{}
	item.ID = id
	err = conn.QueryRow(context.Background(), "select name, description from e_commerce.items where id = $1", item.ID).Scan(&item.Name, &item.Description)
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Println(err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Error there is no item with this id"})
			return
		}

		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get information from the database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"item": item})
}
