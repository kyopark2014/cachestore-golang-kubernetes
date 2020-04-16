package db

import (
	"cachestore-golang-kubernetes/internal/config"
	"cachestore-golang-kubernetes/internal/data"
	"cachestore-golang-kubernetes/internal/log"
	"database/sql"
	"net/http"
	"os"
	"strconv"
)

// MyDb is the main database
var MyDb *sql.DB

// Dbname is the name of database
var Dbname string

// Dbtable is the table name of the database
var Dbtable string

// NewDatabase is initiate the SQL database
func NewDatabase(cfg config.SQLConfig) {
	// Create database
	create, err := MyDb.Query("CREATE DATABASE IF NOT EXISTS " + Dbname)
	if err != nil {
		log.E("Fail to create database %v", err)
		os.Exit(1)
	}
	// be careful deferring Queries if you are using transactions
	defer create.Close()

	// Create Table
	//  UID   string, Name  string, Email string, Age   int
	//  CREATE TABLE IF NOT EXISTS my_db.data (uid VARCHAR(20), name VARCHAR(20), email VARCHAR(30), age BIGINT);
	var statement string = "CREATE TABLE IF NOT EXISTS " + Dbname + "." + Dbtable + " (" + "uid VARCHAR(20), name VARCHAR(20), email VARCHAR(30), age BIGINT" + ")"
	log.D("%v", statement)

	createTable, err := MyDb.Query(statement)
	if err != nil {
		log.E("Fail to create database %v", err)
		os.Exit(1)
	}
	defer createTable.Close()

	log.I("Successfully connected to MySQL database: %v", cfg.Host+":"+cfg.Port)
}

// InsertToDB is to put input data into database
func InsertToDB(value data.UserProfile) error {
	// INSERT INTO my_db.data (uid, name, email, age) VALUES("johnny", "Park", "john@email.com",21);
	statement := "INSERT INTO " + Dbname + "." + Dbtable + " (uid, name, email, age) VALUES (\"" +
		value.UID + "\", \"" + value.Name + "\", \"" + value.Email + "\", " + strconv.FormatInt(int64(value.Age), 10) + ")"
	log.D("%v", statement)

	insert, err := MyDb.Query(statement)
	if err != nil {
		log.E("Fail to insert data %v", err)
		return err
	}
	defer insert.Close()
	log.D("Successfully inserted in SQL database")

	return nil
}

// RetrevefromDB is to get a cached data from redis. If there is no data in redis cache, it will check the database.
func RetrevefromDB(uid string) (data.UserProfile, int) {
	// search in data base
	// SELECT * FROM my_db.data WHERE uid = "kyopark";
	statement := "SELECT * FROM " + Dbname + "." + Dbtable + " WHERE uid = \"" + uid + "\""
	log.D("%v", statement)

	var value data.UserProfile

	results, err := MyDb.Query(statement)
	if err != nil {
		log.E("Fail to retrieve: %v", err)
		return value, http.StatusInternalServerError
	}
	defer results.Close()

	isExist := false
	for results.Next() {
		if err = results.Scan(&value.UID, &value.Name, &value.Email, &value.Age); err != nil {
			log.E("Faill to query: %v", err)
			return value, http.StatusInternalServerError
		}

		log.D("data: %v %v %v %v", value.UID, value.Name, value.Email, value.Age)
		isExist = true
	}

	if isExist {
		return value, 0
	} else {
		log.E("Data is not found in database")
		return value, http.StatusNotFound
	}
}
