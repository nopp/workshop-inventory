-- Create a database if it doesn't exist
CREATE DATABASE IF NOT EXISTS yourdb;

-- Use the created database
USE yourdb;

-- Create a table to store cabinets
CREATE TABLE IF NOT EXISTS cabinets (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

-- Create a table to store shelves
CREATE TABLE IF NOT EXISTS shelves (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    cabinet_id INT NOT NULL,
    FOREIGN KEY (cabinet_id) REFERENCES cabinets(id)
);

-- Create a table to store products
CREATE TABLE IF NOT EXISTS products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    cabinet_id INT NOT NULL,
    shelf_id INT NOT NULL,
    FOREIGN KEY (cabinet_id) REFERENCES cabinets(id),
    FOREIGN KEY (shelf_id) REFERENCES shelves(id)
);
