package main

import (
	"context"
	"log"
	"os"
	"tg_game_wishlist/api/igdb"
	tgClient "tg_game_wishlist/clients/telegram"
	event_consumer "tg_game_wishlist/consumer/event-consumer"
	"tg_game_wishlist/events/telegram"
	"tg_game_wishlist/storage/sqlite"

	"github.com/joho/godotenv"
)

const (
	tgBotHost         = "api.telegram.org"
	batchSize         = 100
	timeout           = 60
	igdbHost          = "api.igdb.com/v4"
	sqliteStoragePath = "storage.db"
)

func init() {
	// loads values from .env into the environment
	if err := godotenv.Load(); err != nil {
		log.Fatal("No .env file found, loading from real environment variables")
	}
}

func main() {
	s, err := sqlite.New(sqliteStoragePath)
	if err != nil {
		log.Fatal("can't connect to storage: ", err)
	}

	if err := s.Init(context.TODO()); err != nil {
		log.Fatal("can't init storage: ", err)
	}

	token := mustEnv("TG_BOT_TOKEN")
	apiClientId := mustEnv("API_CLIENT_ID")
	apiToken := mustEnv("API_TOKEN")
	apiTokenType := mustEnv("API_TOKEN_TYPE")

	client := tgClient.New(tgBotHost, token)

	processor := telegram.NewProcessor(
		client,
		igdb.New(igdbHost, apiClientId, apiToken, apiTokenType),
		s,
	)

	fetcher := telegram.NewFetcher(client)

	consumer := event_consumer.New(fetcher, processor, batchSize, timeout)

	log.Print("service started")

	if err := consumer.Start(); err != nil {
		log.Fatal("service is stopped", err)
	}
}

func mustEnv(envName string) string {
	env := os.Getenv(envName)
	if envName == "" {
		log.Fatal(envName + " is not specified")
	}
	return env
}
