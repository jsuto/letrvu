package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/yourusername/letrvu/internal/api"
	"github.com/yourusername/letrvu/internal/session"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	flag.Parse()

	sessions := session.NewStore()
	handler := api.NewRouter(sessions)

	log.Printf("letrvu listening on %s", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}
