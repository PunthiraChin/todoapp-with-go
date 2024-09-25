package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB has its own data type
type Todo struct {
	ID        primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"` // ให้ MongoDB create random ID ให้
	Completed bool               `json:"completed"`
	Body      string             `json:"body" `
}

var collection *mongo.Collection // Pointer to MongoDB Collection

func main() {
	fmt.Println("Hello World")
	if os.Getenv("ENV") != "production" {
		// Load the .env file if not in production
		err := godotenv.Load(".env")
		if err != nil {
			log.Fatal("Error loading .env file:", err)
		}
	}

	// Connect to MongoDB Database
	MONGODB_URI := os.Getenv("MONGODB_URI")
	clientOptions := options.Client().ApplyURI(MONGODB_URI) // Create a client that connects to MongoDB
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	// Check the connection with "Ping" method
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to MONGODB")

	collection = client.Database("golang_db").Collection("todos")

	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))
	app.Get("/api/todos", getTodos)
	app.Post("/api/todos", createTodos)
	app.Patch("/api/todos/:id", updateTodos)
	app.Delete("/api/todos/:id", deleteTodos)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5001"
	}
	// ถ้าอยู่ที่ production จะเปลี่ยนเป็น static และถ้าไม่ได้เข้ามาที่ route ของ backend >> จะพาไป FE
	if os.Getenv("ENV") == "production" {
		app.Static("/", "./client/dist")
	}
	log.Fatal(app.Listen("0.0.0.0:" + port))
}

func getTodos(c *fiber.Ctx) error {
	var todos []Todo // สร้าง variable ขึ้นมาเตรียมรอรับ get return from the database

	cursor, err := collection.Find(context.Background(), bson.M{}) // Pass no filter เลยไม่ใส่อะไรใน object
	if err != nil {
		return err
	}

	// postpone the function call untile the surrounding function is completed
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var todo Todo
		if err := cursor.Decode(&todo); err != nil {
			return err
		}
		todos = append(todos, todo)
	}
	return c.JSON(todos)

	// cursor: when you execute a query in MongoDB, it returns "cursor"= a pointer to a result set

}

func createTodos(c *fiber.Ctx) error {
	todo := new(Todo) // {id:0, completed:false, body:""}

	// Parase the request body stored in "c" to "todo"
	if err := c.BodyParser(todo); err != nil {
		return err
	}
	if todo.Body == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Todo body cannot be empty"})
	}

	// Insert data to MongoDB
	insertResult, err := collection.InsertOne(context.Background(), todo)
	if err != nil {
		return err
	}
	// Update todo ID with the inserted id
	todo.ID = insertResult.InsertedID.(primitive.ObjectID)
	return c.Status(201).JSON(todo)
}

func updateTodos(c *fiber.Ctx) error {

	id := c.Params("id")
	// This converts the string id to a MongoDB ObjectID. The ObjectIDFromHex function attempts to parse the string as a valid 24-character hexadecimal string.
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid Todo ID"})
	}

	// กำหนด filter เพื่อหา record ที่เราจะ update
	filter := bson.M{"_id": objectID}
	// This defines the update operation. It uses the $set operator to set the completed field to true.
	update := bson.M{"$set": bson.M{"completed": true}}

	_, err = collection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		return err
	}
	return c.Status(200).JSON(fiber.Map{"success": "true"})
}

func deleteTodos(c *fiber.Ctx) error {
	id := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid Todo ID"})
	}
	filter := bson.M{"_id": objectID}
	_, err = collection.DeleteOne(context.Background(), filter)

	if err != nil {
		return err
	}
	return c.Status(200).JSON(fiber.Map{"success": true})
}
