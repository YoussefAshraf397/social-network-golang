package main

import (
	"database/sql"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"social-network/internal/handler"
	"social-network/internal/service"
	"strconv"
)

const ()

func main() {
	godotenv.Load()
	var (
		Port         = env("PORT", "3000")
		Origin       = env("ORIGIN", "http://localhost:"+Port)
		databaseURL  = env("DATABASE_URL", "postgresql://root@127.0.0.1:26257/socialnetwork?sslmode=disable")
		secretKey    = env("SECRET_KEY", "supersecretkeyyoushouldnotcommit")
		smtpHost     = env("SMTP_HOST", "smtp.mailtrap.io")
		smtpPort     = intEnv("SMTP_PORT", 465)
		smtpUsername = mustEnv("SMTP_USERNAME")
		smtpPassword = mustEnv("SMTP_PASSWORD")
	)
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalln("cloud not open database connection: %v\n", err)
		return
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalln("cannot ping to database: %v/n", err)
		return
	}

	s := service.New(service.Config{
		DB:           db,
		Origin:       Origin,
		SecretKey:    secretKey,
		SMTPHost:     smtpHost,
		SMTPPort:     smtpPort,
		SMTPUsername: smtpUsername,
		SMTPPassword: smtpPassword,
	})
	h := handler.New(s)

	log.Println("accepting connection on port %s\n", Port)
	if err = http.ListenAndServe(":"+Port, h); err != nil {
		log.Fatalf("could not start server %v\n", err)
	}
}

func env(key, fallbackValue string) string {
	s, ok := os.LookupEnv(key)
	if !ok {
		return fallbackValue
	}
	return s
}

func mustEnv(key string) string {
	s, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("%s missing on enviroment varaible\n", key)
		return ""
	}
	return s
}

func intEnv(key string, fallbackValue int) int {
	s, ok := os.LookupEnv(key)
	if !ok {
		return fallbackValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return fallbackValue
	}
	return i
}
