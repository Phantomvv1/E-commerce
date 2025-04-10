package authentication

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

const (
	Admin = iota + 1
	User
)

type Profile struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Type  byte   `json:"type"`
}

func GenerateJWT(id int, accountType byte, email string) (string, error) {
	claims := jwt.MapClaims{
		"id":         id,
		"type":       accountType,
		"email":      email,
		"expiration": time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtKey := os.Getenv("JWT_KEY")
	return token.SignedString([]byte(jwtKey))
}

func ValidateJWT(tokenString string) (int, byte, error) {
	claims := &jwt.MapClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.ErrUnsupported
		}

		return []byte(os.Getenv("JWT_KEY")), nil
	})

	if err != nil || !token.Valid {
		return 0, 0, err
	}

	expiration, ok := (*claims)["expiration"].(float64)
	if !ok {
		return 0, 0, errors.New("Error parsing the expiration date of the token")
	}

	if int64(expiration) < time.Now().Unix() {
		return 0, 0, errors.New("Error token has expired")
	}

	id, ok := (*claims)["id"].(float64)
	if !ok {
		return 0, 0, errors.New("Incorrect type of id")
	}

	accountType, ok := (*claims)["type"].(float64)
	if !ok {
		return 0, 0, errors.New("Incorrect type of account")
	}

	return int(id), byte(accountType), nil
}

func SHA512(text string) string {
	algorithm := sha512.New()
	algorithm.Write([]byte(text))
	result := algorithm.Sum(nil)
	return fmt.Sprintf("%x", result)
}

func CreateAuthTable(conn *pgx.Conn) error {
	_, err := conn.Exec(context.Background(), "create table if not exists e_commerce.authentication (id serial primary key, name text, email text, password text, type int)")
	return err
}

func SignUp(c *gin.Context) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Databse connection failed"})
		return
	}

	var information map[string]string
	json.NewDecoder(c.Request.Body).Decode(&information) //name, email, password, type

	if err = CreateAuthTable(conn); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating a tablew for authentication"})
		return
	}

	validEmail, err := regexp.MatchString(".*@.*\\..*", information["email"])
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusForbidden, gin.H{"error": "Error invalid email"})
		return
	}

	if !validEmail {
		log.Println("Invalid email")
		c.JSON(http.StatusForbidden, gin.H{"error": "Error invalid email"})
		return
	}

	var check string
	err = conn.QueryRow(context.Background(), "select email from e_commerce.authentication where email = $1", information["email"]).Scan(&check)
	emailExists := true
	if err != nil {
		if err == pgx.ErrNoRows {
			emailExists = false
		} else {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting the password from the table"})
			return
		}
	}

	if emailExists {
		log.Println(err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "There is already a person with this email"})
		return
	}

	accountTypeStr, ok := information["type"]
	if !ok {
		log.Println("Incorrectly provided type of account")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided type of account"})
		return
	}

	accountType := 0
	if accountTypeStr == "admin" {
		accountType = Admin
	} else {
		accountType = User
	}

	hashedPassword := SHA512(information["password"])
	_, err = conn.Exec(context.Background(), "insert into e_commerce.authentication (name, email, password, type) values ($1, $2, $3, $4)",
		information["name"], information["email"], hashedPassword, accountType)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error inserting the information into the database."})
		return
	}

	c.JSON(http.StatusOK, nil)
}

func LogIn(c *gin.Context) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	if err = CreateAuthTable(conn); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	var information map[string]string
	json.NewDecoder(c.Request.Body).Decode(&information) //email, password

	var passwordCheck, name, email string
	var accoutType byte
	var id int
	err = conn.QueryRow(context.Background(), "select id, password, name, type, email from e_commerce.authentication a where a.email = $1;", information["email"]).Scan(
		&id, &passwordCheck, &name, &accoutType, &email)
	if err != nil {
		if err == pgx.ErrNoRows {
			log.Println(err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "There isn't anybody registered with this email!"})
			return
		} else {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while trying to log in"})
			return
		}
	}

	if SHA512(information["password"]) != passwordCheck {
		log.Println("Wrong password")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error wrong password"})
		return
	}

	jwtToken, err := GenerateJWT(id, accoutType, email)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error while generating your token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": jwtToken})
}

func GetCurrentProfile(c *gin.Context) {
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error couldn't connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	var tokenString map[string]string
	json.NewDecoder(c.Request.Body).Decode(&tokenString)

	id, accountType, err := ValidateJWT(tokenString["token"])
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error validating the token"})
		return
	}

	var name, email string
	err = conn.QueryRow(context.Background(), "select name, email from e_commerce.authentication where id = $1", id).Scan(&name, &email)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting information from the database"})
		return
	}

	userProfile := Profile{
		ID:    id,
		Name:  name,
		Email: email,
		Type:  accountType,
	}

	c.JSON(http.StatusOK, gin.H{"profile": userProfile})
}

func GetAllUsers(c *gin.Context) {
	var information map[string]string
	json.NewDecoder(c.Request.Body).Decode(&information) // token

	token, ok := information["token"]
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
		c.JSON(http.StatusForbidden, gin.H{"error": "Error only admins can view all of the accounts"})
		return
	}

	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to connect to the database"})
		return
	}
	defer conn.Close(context.Background())

	rows, err := conn.Query(context.Background(), "select id, name, email, type from e_commerce.authentication")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error couldn't get information from the database"})
		return
	}

	var profiles []Profile
	for rows.Next() {
		profile := Profile{}
		err = rows.Scan(&profile.ID, &profile.Name, &profile.Email, &profile.Type)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error working with the data from the database"})
			return
		}

		profiles = append(profiles, profile)
	}

	if rows.Err() != nil {
		log.Println(rows.Err())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error working with the data from the database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profiles": profiles})
}
