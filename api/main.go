package main

import (
	"fmt"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/siddarth99/banking-ledger/api/handlers"
	internal "github.com/siddarth99/banking-ledger/pkg"
)

func main() {
	router := gin.Default()

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Create RabbitMQ connection
	aqmpConn, err := internal.CreateAMQPConnection(
		"amqp://" +
			os.Getenv("RABBITMQ_USER") + ":" +
			os.Getenv("RABBITMQ_PASSWORD") + "@" +
			os.Getenv("RABBITMQ_HOST") + ":" +
			os.Getenv("RABBITMQ_PORT") + "/")
	if err != nil {
		panic(err)
	}

	defer internal.CloseAMQPConnection(aqmpConn)

	ctx := gin.Context{}

	amqpChannel, err := aqmpConn.Channel()
	if err != nil {
		panic(err)
	}

	defer internal.CloseAMQPChannel(amqpChannel)

	// Declare queue
	createAccountQueue, err := internal.QueueDeclare(amqpChannel, "account_creator", true, false, false, false)
	if err != nil {
		panic(err)
	}
	router.POST("/createAccount", handlers.CreateAccountHandler(&ctx, amqpChannel, createAccountQueue.Name))

	// Declare queue
	transactionQueue, err := internal.QueueDeclare(amqpChannel, "transaction_processor", true, false, false, false)
	if err != nil {
		panic(err)
	}

	router.POST("/transact", handlers.TransactionHandler(&ctx, amqpChannel, transactionQueue.Name))

	// Initialize Elasticsearch client
	esConfig := elasticsearch.Config{
		Addresses: []string{os.Getenv("ELASTICSEARCH_URL")},
	}

	esClient, err := internal.NewElasticsearchClient(esConfig)
	if err != nil {
		panic(fmt.Sprintf("Error creating Elasticsearch client: %s", err))
	}

	// Test the connection
	res, err := esClient.Info()
	if err != nil {
		panic(fmt.Sprintf("Error getting Elasticsearch info: %s", err))
	}
	defer res.Body.Close()

	router.GET("/account/:accountNumber/transactionHistory", handlers.GetTransactionHistoryHandler(esClient))

	router.GET("/account/status/:referenceId", handlers.GetAccountStatusHandler(esClient))

	router.Run(":8080")
}
