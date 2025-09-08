package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	amqpc "CommandHandler/config/amqp"
	dbcfg "CommandHandler/config/db"
	"CommandHandler/services"
	"CommandHandler/services/consumer"
	"CommandHandler/services/dispatcher"
	"CommandHandler/utils"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	log := utils.NewLogger()
	cfg := dbcfg.Load()

	// SIGINT/SIGTERM → graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		log.Warn("shutdown signal received")
		cancel()
	}()

	// 1) Connect DB
	sqlDB, _, err := dbcfg.ConnectAny(ctx, cfg)
	if err != nil {
		log.Fatal("MSSQL connect failed", "err", err)
	}
	defer sqlDB.Close()
	log.OK("SQL Server connected", "server", cfg.DBServer, "db", cfg.DBName)

	// 2) Resolve StoreID
	storeID, err := dbcfg.GetStoreID(ctx, sqlDB)
	if err != nil {
		log.Fatal("Resolve StoreID failed", "err", err)
	}
	log.OK("StoreID resolved", "store_id", storeID)

	// 3) Connect RabbitMQ
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}
	rmq := amqpc.NewClient(log)
	if err := rmq.Connect(ctx, rabbitURL); err != nil {
		log.Fatal("RabbitMQ connect failed", "err", err)
	}
	defer rmq.Close()
	log.OK("RabbitMQ connected")

	// 4) Setup exchange/queue/binding
	queue, key, err := rmq.SetupRepairQueue(ctx, storeID)
	if err != nil {
		log.Fatal("Queue binding failed", "err", err)
	}
	log.OK("Queue bound", "queue", queue, "key", key)

	// 5) Build services & dispatcher, lalu start consumer (blocking)
	svc := services.New(sqlDB)
	h := dispatcher.New(log, rmq.Channel(), svc)

	if err := consumer.Start(ctx, log, rmq.Channel(), queue, h); err != nil && ctx.Err() == nil {
		log.Fatal("consumer stopped", "err", err)
	}
	// kalau ctx selesai (SIGINT/SIGTERM), fungsi di atas return dengan ctx.Err()
}
