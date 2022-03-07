package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html"
	"github.com/gofiber/websocket/v2"
	"github.com/lichtwellenreiter/wowebtest/persistence"
	"log"
	"os"
)

type client struct{} // Add more data to this type if needed

var mg persistence.MongoInstance

var (
	port       = getEnv("PORT", "3000")
	clients    = make(map[*websocket.Conn]client) // Note: although large maps with pointer-like types (e.g. strings) as keys are slow, using pointers themselves as keys is acceptable and fast
	register   = make(chan *websocket.Conn)
	broadcast  = make(chan string)
	unregister = make(chan *websocket.Conn)
)

func main() {

	engine := html.New("./views", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	setupRoutes(app)
	setupApp(app)

	// Connect to the database
	mi, err := persistence.Connect()
	if err != nil {
		log.Fatal("Cannot connect to DB: ", err)
	}

	mg = persistence.MongoInstance{
		Client: mi.Client,
		Db:     mi.Db,
	}

	go runHub()

	log.Fatal(app.Listen(fmt.Sprintf(":%v", port)))
}

func setupApp(app *fiber.App) {
	app.Static("/", "./views")

	app.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${pid} ${status} - ${method} ${path}\n",
	}))

	app.Use(func(c *fiber.Ctx) error {
		// Returns true if the client requested upgrade to the WebSocket protocol
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return c.SendStatus(fiber.StatusUpgradeRequired)
	})
}

func setupRoutes(app *fiber.App) {

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("home", nil)
	})

	app.Get("/document", func(c *fiber.Ctx) error {
		return c.Render("document", nil)
	})

	app.Get("/panic", func(c *fiber.Ctx) error {
		panic("Just a test")
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		// When the function returns, unregister the client and close the connection
		defer func() {
			unregister <- c
			c.Close()
		}()

		// Register the client
		register <- c

		for {
			messageType, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Println("read error:", err)
				}

				return // Calls the deferred function, i.e. closes the connection on error
			}

			if messageType == websocket.TextMessage {
				// Broadcast the received message
				broadcast <- string(message)
			} else {
				log.Println("websocket message received of type", messageType)
			}
		}
	}))

}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return defaultValue
	}
	return value
}

func runHub() {
	for {
		select {
		case connection := <-register:
			clients[connection] = client{}
			log.Println("connection registered")

		case message := <-broadcast:
			log.Println("message received:", message)

			// Send the message to all clients
			for connection := range clients {
				if err := connection.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
					log.Println("write error:", err)

					connection.WriteMessage(websocket.CloseMessage, []byte{})
					connection.Close()
					delete(clients, connection)
				}
			}

		case connection := <-unregister:
			// Remove the client from the hub
			delete(clients, connection)

			log.Println("connection unregistered")
		}
	}
}
