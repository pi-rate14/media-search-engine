package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type app struct {
	gpt GPT
}

func NewConfig() *app {
	_ = godotenv.Load(".env")
	gpt := NewGPT(os.Getenv("OPEN_AI_KEY"))
	return &app{
		gpt: *gpt,
	}
}

func main() {
	app := NewConfig()
	http.HandleFunc("/query", app.QueryGPT)
	err := http.ListenAndServe(":8086", nil)
	if err != nil {
		log.Fatal(err)
	}
}
