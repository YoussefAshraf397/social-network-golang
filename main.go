package main

import (
	"database/sql"
	"github.com/hako/branca"
	_ "github.com/jackc/pgx/stdlib"
	"log"
	"net/http"
	"os"
	"social-network/internal/handler"
	"social-network/internal/service"
)

const ()

func main() {
	var (
		Port        = env("PORT", "3000")
		Origin      = env("ORIGIN", "http://localhost:"+Port)
		databaseURL = env("DATABASE_URL", "postgresql://root@127.0.0.1:26257/socialnetwork?sslmode=disable")
		Branka_Key  = env("BRANCA_KEY", "supersecretkeyyoushouldnotcommit")
	)
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalln("cloud not open database connection: %v\n", err)
		return
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalln("cannot ping to database: %v/n", err)
		return
	}

	codec := branca.NewBranca(Branka_Key)
	codec.SetTTL(uint32(service.TokenLifeSpan.Seconds()))
	s := service.New(db, codec, Origin)
	h := handler.New(s)

	log.Println("accepting connection on port %s\n", Port)
	if err = http.ListenAndServe(":"+Port, h); err != nil {
		log.Fatalf("could not start server %v\n", err)
	}
}

func env(key, fallbackValue string) string {
	s := os.Getenv(key)
	if s == "" {
		return fallbackValue
	}
	return s
}
