package emails

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	. "github.com/Phantomvv1/E-commerce/internal/authentication"
	"github.com/gin-gonic/gin"
	"gopkg.in/gomail.v2"
)

func SendEmail(c *gin.Context) {
	var information map[string]string
	json.NewDecoder(c.Request.Body).Decode(&information) // token && subject && text

	token, ok := information["token"]
	if !ok {
		log.Println("Incorrectly provided token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided token"})
		return
	}

	_, _, err := ValidateJWT(token)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error invalid token"})
		return
	}

	email, err := GetEmail(token)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error couldn't parse your token correctly"})
		return
	}

	log.Println(email)

	subject, ok := information["subject"]
	if !ok {
		log.Println("Incorrectly provided subject of the email")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error incorrectly provided subject of the email"})
		return
	}

	text, ok := information["text"]
	if !ok {
		log.Println("Incorrectly provided text for the email")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error Incorrectly provided text for the email"})
		return
	}

	m := gomail.NewMessage()
	m.SetHeader("From", os.Getenv("SMTP_USERNAME"))
	m.SetHeader("To", email)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", text)

	dialer := gomail.NewDialer(os.Getenv("SMTP_FROM"), 465, os.Getenv("SMTP_USERNAME"), os.Getenv("SMTP_PASSWORD"))

	if err = dialer.DialAndSend(m); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error unable to send the email to the person"})
		return
	}

	c.JSON(http.StatusOK, nil)
}
