package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"mongo-golang/controllers"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	client := getMongoClient()
	defer client.Disconnect(context.Background())

	router := httprouter.New()
	uc := controllers.NewUserController(client)

	router.GET("/user/:id", uc.GetUser)
	router.POST("/user", uc.CreateUser)
	router.DELETE("/user/:id", uc.DeleteUser)

	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func getMongoClient() *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("MongoDB not reachable:", err)
	}

	return client
}
