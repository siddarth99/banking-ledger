package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"

	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/jackc/pgx/v5"
	internal "github.com/siddarth99/banking-ledger/pkg"
	"github.com/siddarth99/banking-ledger/workers/processor"
)

func main() {
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

	amqpChannel, err := aqmpConn.Channel()
	if err != nil {
		panic(err)
	}

	defer internal.CloseAMQPChannel(amqpChannel)

	// Declare queue
	queue, err := internal.QueueDeclare(amqpChannel, os.Getenv("RABBITMQ_QUEUE_NAME"), true, false, false, false)
	if err != nil {
		panic(err)
	}

	msgsChan, err := amqpChannel.Consume(
		queue.Name, // queueConsume
		"",         // consumer
		false,       // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)

	if err != nil {
		log.Printf("Failed to register a consumer: %s", err)
		panic(err)
	}

	// urlExample := "postgres://username:password@localhost:5432/database_name"
	conn, err := pgx.Connect(context.Background(), fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME")))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	defer conn.Close(context.Background())

	// Initialize Elasticsearch client
	esConfig := elasticsearch.Config{
		Addresses: []string{os.Getenv("ELASTICSEARCH_URL")},
	}

	esClient, err := internal.NewElasticsearchClient(esConfig)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Elasticsearch client: %v\n", err)
		os.Exit(1)
	}

	// Test the connection
	res, err := esClient.Info()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Elasticsearch: %v\n", err)
		os.Exit(1)
	}
	defer res.Body.Close()

	log.Println("Successfully connected to Elasticsearch")

	wg := sync.WaitGroup{}

	wg.Add(1)
	
	// Start 4 worker goroutines
	numWorkers := 4 // Default value
	if workerCount, err := strconv.Atoi(os.Getenv("NUM_WORKERS")); err == nil && workerCount > 0 {
		numWorkers = workerCount
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int, waitGroup *sync.WaitGroup) {
			defer waitGroup.Done()
			log.Printf("Worker %d started", workerID)
			
			for d := range msgsChan {
				log.Printf("Worker %d received a message: %s", workerID, d.Body)
				
				switch os.Getenv("RABBITMQ_QUEUE_NAME") {
				case "account_creator":
					var accountInfo processor.AccountData
					
					err := json.Unmarshal(d.Body, &accountInfo)
					
					if err != nil {
						log.Printf("Error: %s\n", err)
						d.Ack(false)
						continue
					}
					
					processWorker := processor.CreateAccountProcessor{
						ProcessWorker: processor.ProcessWorker{
							PgxConn: conn,
							EsConn:  esClient,
						},
						Data: accountInfo,
					}
					
					err = processWorker.CreateAccount(context.Background())
					
					if err != nil {
						log.Println(err)
					}
					d.Ack(false)
				case "transaction_processor":
					var transactionInfo processor.TransactionData
					err := json.Unmarshal(d.Body, &transactionInfo)
					
					if err != nil {
						log.Println(err)
						d.Ack(false)
						continue
					}
					
					processWorker := processor.TransactionProcessor{
						ProcessWorker: processor.ProcessWorker{
							PgxConn: conn,
							EsConn:  esClient,
						},
						Data: transactionInfo,
					}
					
					err = processWorker.ProcessTransaction(context.Background())
					if err != nil {
						log.Println(err)
					}
					d.Ack(false)
				}
			}
		}(i, &wg)
	}

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")

	wg.Wait()
}
