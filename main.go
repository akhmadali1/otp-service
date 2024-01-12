package main

import (
	"bola-wa-service/routes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var client *whatsmeow.Client

func main() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:otpdbtemp.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client = whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	router := routes.SetupRoutes(client)
	router.Run(":8073")
	go monitorConnection()

	// Use a buffered channel for signals to prevent SA1017 warning
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Use a simple channel receive instead of select with a single case (S1000)
	sig := <-signalChan
	fmt.Printf("Received signal: %v\n", sig)
	client.Disconnect()
	os.Exit(0)
}

func monitorConnection() {
	for {
		time.Sleep(time.Second * 10) // Adjust the interval based on your requirements
		if !client.IsConnected() {
			fmt.Println("Connection lost. Restarting service...")
			restartService()
		}
	}
}

func restartService() {
	cmd := exec.Command("sudo", "service", "bola-wa-service", "restart")
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error restarting service:", err)
	}
}
