package items

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	. "github.com/Phantomvv1/E-commerce/internal/authentication"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Item struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float32 `json:"price"`
}

func CreateItemsTable(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), "create table if not exists e_commerce.items (id serial primary key, name text, description text, price numeric)")
	return err
}

func CreateItem(c *gin.Context) {
	var information map[string]interface{}
	json.NewDecoder(c.Request.Body).Decode(&information) // token && name && description && price

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

	price, ok := information["price"].(float64)
	if !ok {
		log.Println("Incorrectly provided price of the item")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided price of the item"})
		return
	}

	_, err = conn.Exec(context.Background(), "insert into e_commerce.items (name, description, price) values ($1, $2, $3)", name, desc, price)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to put the information about the item in the database"})
		return
	}

	c.JSON(http.StatusOK, nil)
}

func UpdateItem(c *gin.Context) {
	var information map[string]interface{}
	json.NewDecoder(c.Request.Body).Decode(&information) // token && id && (name || desc, || price)

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
	if !ok {
		updateDesc = false
	}

	updatePrice := true
	price, ok := information["price"].(float64)
	if !ok && !updateDesc && !updateName {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error not enough information to update the item with"})
		return
	} else if !ok {
		updatePrice = false
	}

	if updateName {
		_, err = conn.Exec(context.Background(), "update e_commerce.items set name = $1 where id = $2", name, itemID)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to update the information in the database"})
			return
		}
	}

	if updateDesc {
		_, err = conn.Exec(context.Background(), "update e_commerce.items set description = $1 where id = $2", desc, itemID)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to update the information in the database"})
			return
		}

	}

	if updatePrice {
		_, err = conn.Exec(context.Background(), "update e_commerce.items set price = $1 where id = $2", price, itemID)
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
	err = conn.QueryRow(context.Background(), "select name, description, price from e_commerce.items where id = $1", item.ID).Scan(&item.Name, &item.Description, &item.Price)
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

func SearchForItem(c *gin.Context) {
	var information map[string]string
	json.NewDecoder(c.Request.Body).Decode(&information)

	name, ok := information["name"]
	if !ok {
		log.Println("Incorrectly provided name of the item")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided name of the item"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), "select id, name, description, price from e_commerce.items i where i.name ~ $1", name)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get information from the database"})
		return
	}

	var items []Item
	for rows.Next() {
		item := Item{}
		err = rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error working with the information from the database"})
			return
		}

		items = append(items, item)
	}

	if rows.Err() != nil {
		log.Println(rows.Err())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error working with the information from the database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func GetAllItems(c *gin.Context) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), "select id, name, description, price from e_commerce.items order by id")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get information from the database"})
		return
	}

	var items []Item
	for rows.Next() {
		item := Item{}
		err = rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error working with the items"})
			return
		}

		items = append(items, item)
	}

	if rows.Err() != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error working with the items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func GetRandomItem(c *gin.Context) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	item := Item{}
	err = conn.QueryRow(context.Background(), "select id, name, description, price from e_commerce.items limit 1").Scan(&item.ID, &item.Name, &item.Description, &item.Price)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get information from the database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"item": item})
}

func DeleteItem(c *gin.Context) {
	var information map[string]interface{}
	json.NewDecoder(c.Request.Body).Decode(&information) // token && id

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
		c.JSON(http.StatusForbidden, gin.H{"error": "Error only admins can delete items"})
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
		log.Println("Incorrectly provided id of the item")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided id of the item"})
		return
	}

	check := 0
	err = conn.QueryRow(context.Background(), "delete from e_commerce.items where id = $1 returning id", id).Scan(&check)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Error there is no item with this id"})
			return
		}

		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to delete the item from the database"})
		return
	}

	c.JSON(http.StatusOK, nil)
}

func CountItems(c *gin.Context) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	count := 0
	err = conn.QueryRow(context.Background(), "select count(*) from e_commerce.items").Scan(&count)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to get information from the database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}
