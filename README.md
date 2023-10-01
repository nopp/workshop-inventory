# home workshop

**1. Get a List of Products:**

```bash
curl -X GET http://localhost:8080/products
```

**2. Get Product Details by ID:**

Replace `<product_id>` with the actual product ID you want to retrieve.

```bash
curl -X GET http://localhost:8080/product/<product_id>
```

**3. Create a New Product:**

```bash
curl -X POST -H "Content-Type: application/json" -d '{
  "name": "New Product",
  "cabinet": 1,
  "shelf": 2
}' http://localhost:8080/product
```

This command sends a POST request with JSON data to create a new product.

**4. Update a Product:**

Replace `<product_id>` with the actual product ID you want to update.

```bash
curl -X PUT -H "Content-Type: application/json" -d '{
  "name": "Updated Product",
  "cabinet": 3,
  "shelf": 4
}' http://localhost:8080/product/<product_id>
```

This command sends a PUT request with JSON data to update an existing product.

**5. Delete a Product:**

Replace `<product_id>` with the actual product ID you want to delete.

```bash
curl -X DELETE http://localhost:8080/product/<product_id>
```

This command sends a DELETE request to remove a product from the database.

Ensure that the GoLang API is running and listening on `http://localhost:8080` as specified in the previous GoLang example. Adjust the URL and data in the `curl` commands to match your API endpoints and test cases.
