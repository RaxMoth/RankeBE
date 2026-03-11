package server
// Fiber app wiring, middleware registration, graceful shutdown
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"github.com/gofiber/fiber/v2"
)

func main() {

















}	}		log.Fatalf("Error starting server: %v", err)	if err := app.Listen(":8080"); err != nil {	}()		_ = app.Shutdown()		log.Println("Shutting down...")		<-c	go func() {	signal.Notify(c, os.Interrupt, syscall.SIGTERM)	c := make(chan os.Signal, 1)	// Graceful shutdown	// Register routes here	// Register middleware here	app := fiber.New()