package main

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/template/html"
	"github.com/gofiber/websocket/v2"
	"github.com/lichtwellenreiter/wowebtest/persistence"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

type client struct{} // Add more data to this type if needed

var mg persistence.MongoInstance

var (
	port       = getEnv("PORT", "3001")
	clients    = make(map[*websocket.Conn]client) // Note: although large maps with pointer-like types (e.g. strings) as keys are slow, using pointers themselves as keys is acceptable and fast
	register   = make(chan *websocket.Conn)
	broadcast  = make(chan string)
	unregister = make(chan *websocket.Conn)
	zaplog     *zap.SugaredLogger
)

type UiElement struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Src      string `json:"src,omitempty"`
	CssClass string `json:"cssclass,omitempty"`
}

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
		//log.Fatal(err.Error())
		zaplog.Error(err.Error())
	}

	mg = persistence.MongoInstance{
		Client: mi.Client,
		Db:     mi.Db,
	}

	go runHub()

	//log.Fatal(app.Listen(fmt.Sprintf(":%v", port)))
	zaplog.Fatal(app.Listen(fmt.Sprintf(":%v", port)))
}

func setupApp(app *fiber.App) {
	app.Static("/", "./views")

	logInit(false)
	zaplog.Sync()
	zaplog.Info("Init Logger")

	/*
		app.Use(logger.New(logger.Config{
			Format: "[${ip}]:${port} ${pid} ${status} - ${method} ${path}\n",
		}))
	*/
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost, http://127.0.0.1",
		AllowHeaders: "Origin, Content-Type, Accept",
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

	app.Get("/jsonuielement", func(c *fiber.Ctx) error {

		query := bson.D{{}}

		cursor, err := mg.Db.Collection("jsonui").Find(c.Context(), query)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		var jsonelements []UiElement = make([]UiElement, 0)

		if err := cursor.All(c.Context(), &jsonelements); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		/*testElement := UiElement{
			ID:       "123456",
			Type:     "div",
			Text:     "This is a test div we do it now from database",
			CssClass: "testdiv",
		}*/
		return c.JSON(jsonelements)
	})

	app.Get("/jsonui", func(c *fiber.Ctx) error {
		return c.Render("jsonui", nil)
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
					//log.Println("read error:", err)
					zaplog.Info("read error:", err)

				}

				return // Calls the deferred function, i.e. closes the connection on error
			}

			if messageType == websocket.TextMessage {
				// Broadcast the received message
				broadcast <- string(message)
			} else {
				//log.Println("websocket message received of type", messageType)
				zaplog.Info("websocket message received of type", messageType)

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
			//log.Println("connection registered")
			zaplog.Info("connection registered")

		case message := <-broadcast:
			//log.Println("message received:", message)
			zaplog.Info("message received:", message)

			// Send the message to all clients
			for connection := range clients {
				if err := connection.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
					//log.Println("write error:", err)
					zaplog.Info("write error:", err)

					connection.WriteMessage(websocket.CloseMessage, []byte{})
					connection.Close()
					delete(clients, connection)
				}
			}

		case connection := <-unregister:
			// Remove the client from the hub
			delete(clients, connection)

			zaplog.Info("connection unregistered")
		}
	}
}

func logInit(d bool) {

	f, err := os.OpenFile("tmp/logfile.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		fmt.Println(err)
	}
	pe := zap.NewProductionEncoderConfig()

	fileEncoder := zapcore.NewJSONEncoder(pe)

	pe.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(pe)

	level := zap.InfoLevel
	if d {
		level = zap.DebugLevel
	}

	core := zapcore.NewTee(
		zapcore.NewCore(fileEncoder, zapcore.AddSync(f), level),
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), level),
	)

	l := zap.New(core)

	zaplog = l.Sugar()
}
