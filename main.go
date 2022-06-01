package main

import (
	"database/sql"
	"fmt"
	"github.com/hako/branca"
	_ "github.com/jackc/pgx/stdlib"
	"log"
	"net/http"
	"social-network/internal/handler"
	"social-network/internal/service"
)

const (
	databaseUrl = "postgresql://root@127.0.0.1:26257/socialnetwork?sslmode=disable"
	port        = 3000
)

func main() {
	db, err := sql.Open("pgx", databaseUrl)
	if err != nil {
		log.Fatalln("cloud not open database connection: %v\n", err)
		return
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalln("cannot ping to database: %v/n", err)
		return
	}

	codec := branca.NewBranca("supersecretkeyyoushouldnotcommit")
	codec.SetTTL(uint32(service.TokenLifeSpan.Seconds()))
	s := service.New(db, codec)
	h := handler.New(s)

	addr := fmt.Sprintf(":%d", port)
	log.Println("accepting connection on port %d\n", port)
	if err = http.ListenAndServe(addr, h); err != nil {
		log.Fatalf("could not start server %v\n", err)
	}
}
