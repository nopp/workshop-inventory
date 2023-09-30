package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

type Product struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Cabinet       string `json:"cabinet"`
	Shelf         string `json:"shelf"`
	ShelfPosition int    `json:"shelfposition"`
}

func Connect() *sql.DB {
	db, err := sql.Open("mysql", "root:123456@tcp(kvm.carlosmalucelli.com:3306)/homeapp")
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func main() {
	router := gin.Default()
	router.GET("/products", getProducts)
	router.GET("/product/:id", getProduct)
	router.POST("/product", createProduct)
	router.PUT("/product/:id", updateProduct)
	router.DELETE("/product/:id", deleteProduct)

	router.Run(":8080")
}

func getProducts(c *gin.Context) {
	db := Connect()
	defer db.Close()
	rows, err := db.Query("SELECT p.id, p.name, c.name, s.name, s.position FROM products as p, shelves as s, cabinets as c WHERE p.shelf_id = s.id and p.cabinet_id = c.id")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var product Product
		err := rows.Scan(&product.ID, &product.Name, &product.Cabinet, &product.Shelf, &product.ShelfPosition)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		products = append(products, product)
	}

	c.JSON(http.StatusOK, products)
}

func getProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	db := Connect()
	defer db.Close()

	var product Product
	err = db.QueryRow("SELECT p.id, p.name, c.name, s.name, s.position FROM products as p, shelves as s, cabinets as c WHERE p.id=? AND p.shelf_id = s.id AND p.cabinet_id = c.id", id).Scan(&product.ID, &product.Name, &product.Cabinet, &product.Shelf, &product.ShelfPosition)
	if err != nil {
		fmt.Print(err.Error())
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, product)
}

func createProduct(c *gin.Context) {
	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := Connect()
	defer db.Close()

	result, err := db.Exec("INSERT INTO products (name, cabinet, shelf) VALUES (?, ?, ?)", product.Name, product.Cabinet, product.Shelf)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	productID, _ := result.LastInsertId()
	product.ID = int(productID)
	c.JSON(http.StatusCreated, product)
}

func updateProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := Connect()
	defer db.Close()

	_, err = db.Exec("UPDATE products SET name=?, cabinet=?, shelf=? WHERE id=?", product.Name, product.Cabinet, product.Shelf, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	product.ID = id
	c.JSON(http.StatusOK, product)
}

func deleteProduct(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	db := Connect()
	defer db.Close()

	_, err = db.Exec("DELETE FROM products WHERE id=?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}
