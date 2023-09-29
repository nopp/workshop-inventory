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

var db *sql.DB

func main() {
    // Replace with your MySQL database configuration.
    db, err := sql.Open("mysql", "username:password@tcp(localhost:3306)/yourdb")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    router := gin.Default()
    router.GET("/products", getProducts)
    router.GET("/product/:id", getProduct)
    router.POST("/product", createProduct)
    router.PUT("/product/:id", updateProduct)
    router.DELETE("/product/:id", deleteProduct)

    router.Run(":8080")
}

type Product struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Cabinet  int    `json:"cabinet"`
    Shelf    int    `json:"shelf"`
}

func getProducts(c *gin.Context) {
    rows, err := db.Query("SELECT * FROM products")
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    var products []Product
    for rows.Next() {
        var product Product
        err := rows.Scan(&product.ID, &product.Name, &product.Cabinet, &product.Shelf)
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

    var product Product
    err = db.QueryRow("SELECT * FROM products WHERE id=?", id).Scan(&product.ID, &product.Name, &product.Cabinet, &product.Shelf)
    if err != nil {
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

    _, err = db.Exec("DELETE FROM products WHERE id=?", id)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusNoContent, nil)
}
