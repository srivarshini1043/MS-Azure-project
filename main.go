package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/secrets"
	"github.com/gorilla/mux"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

type Book struct {
	gorm.Model
	BookName string  `json:"book_name,omitempty"`
	Author   string  `json:"author,omitempty"`
	Price    float64 `json:"price,omitempty"`
}

var DB *gorm.DB
var err error

func initDB() {
	// Set up context and credentials
	ctx := context.Background()
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("failed to get a credential: %v", err)
	}

	// Create a Key Vault client
	client, err := secrets.NewClient("https://sqlkeyvaultdb.vault.azure.net/", cred, nil)
	if err != nil {
		log.Fatalf("failed to create key vault client: %v", err)
	}

	// Retrieve the secret (password)
	secretResp, err := client.GetSecret(ctx, "sqlkeysecretdb", nil)
	if err != nil {
		log.Fatalf("failed to get secret: %v", err)
	}

	password := *secretResp.Value

	// Construct the DSN
	dsn := fmt.Sprintf("sqlserver://azureuser:%s@project-sql-server1.database.windows.net:1433?database=projectdb", password)

	// Connect to the database
	DB, err = gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
}

func GetBooks(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var books []Book
	result := DB.Find(&books)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(books)
}

func GetBook(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}
	var book Book
	result := DB.First(&book, id)
	if result.Error != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(book)
}

func CreateBook(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var book Book
	err := json.NewDecoder(r.Body).Decode(&book)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result := DB.Create(&book)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(book)
}

func UpdateBook(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}
	var book Book
	result := DB.First(&book, id)
	if result.Error != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	err = json.NewDecoder(r.Body).Decode(&book)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	book.ID = uint(id)
	result = DB.Save(&book)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(book)
}

func DeleteBook(w http.ResponseWriter, r *http.Request) {
	if DB == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		http.Error(w, "Invalid ID format", http.StatusBadRequest)
		return
	}

	var book Book
	result := DB.First(&book, id)
	if result.Error != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}

	result = DB.Delete(&book, id)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode("The book is deleted successfully!")
}

func main() {
	port := "8082"
	var initDB bool
	flag.BoolVar(&initDB, "initDB", false, "Initialize the database")
	flag.Parse()

	initDB() // Call to initialize the database connection

	if initDB {
		DB.AutoMigrate(&Book{})
	}

	router := mux.NewRouter()
	router.HandleFunc("/books", GetBooks).Methods("GET")
	router.HandleFunc("/book/{id:[0-9]+}", GetBook).Methods("GET")
	router.HandleFunc("/books", CreateBook).Methods("POST")
	router.HandleFunc("/book/{id:[0-9]+}", UpdateBook).Methods("PUT")
	router.HandleFunc("/book/{id:[0-9]+}", DeleteBook).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":"+port, router))
}
