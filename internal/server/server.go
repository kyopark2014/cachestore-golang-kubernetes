package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"cachestore-golang-kubernetes/internal/config"
	"cachestore-golang-kubernetes/internal/data"
	"cachestore-golang-kubernetes/internal/log"
	"cachestore-golang-kubernetes/internal/mysql"
	"cachestore-golang-kubernetes/internal/rediscache"

	"github.com/gorilla/mux"
)

// Insert is the api to append an Item
func Insert(w http.ResponseWriter, r *http.Request) {
	// parse the data
	var value data.UserProfile
	_ = json.NewDecoder(r.Body).Decode(&value)
	log.D("value: %+v", value)

	// Insert into database
	err := mysql.InsertToDB(value)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	key := value.UID // UID to identify the profile

	_, rErr := rediscache.SetCache(key, &value)
	if rErr != nil {
		log.E("Error of setCache: %v", rErr)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	log.D("Successfully inserted in redis cache")

	w.WriteHeader(http.StatusOK)
}

// Retrieve is the api to search an Item
func Retrieve(w http.ResponseWriter, r *http.Request) {
	uid := strings.Split(r.URL.Path, "/")[2]
	log.D("Looking for uid: %v ...", uid)

	// search in redis cache
	cache, err := rediscache.GetCache(uid)
	if err != nil {
		log.E("Error: %v", err)
	}
	if cache != nil {
		log.D("value from cache: %+v", cache)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cache)
	} else {
		log.D("No data in redis cache then search it in database.")

		// search in database
		value, errCode := mysql.RetrevefromDB(uid)
		if errCode == http.StatusInternalServerError || errCode == http.StatusNotFound {
			w.WriteHeader(errCode)
			return
		} else {
			log.D("Successfully quaried in database: %+v", value)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(value)
	}
}

// LiveCheck is the api to check the pod is alive
func LiveCheck(w http.ResponseWriter, r *http.Request) {
	log.D("Live Check ...")
}

// InitServer initializes the REST api server
func InitServer(conf *config.AppConfig) error {
	// DSN: Data Source Name
	var DSN string = conf.SQL.Username + ":" + conf.SQL.Password + "@" + conf.SQL.Protocol + "(" + conf.SQL.Host + ":" + conf.SQL.Port + ")" + "/"
	log.D("DSN: %v", DSN)

	mysql.Dbname = conf.SQL.Database
	mysql.Dbtable = "data"

	// db, err := sql.Open("mysql", "root:password@tcp(172.17.0.2:3306)/")
	var sqlerr error
	mysql.MyDb, sqlerr = sql.Open("mysql", DSN)
	// if there is an error opening the connection, handle it
	if sqlerr != nil {
		return sqlerr
	}
	// defer the close till after the main function has finished
	// executing
	defer mysql.MyDb.Close()

	// Initiate the SQL database
	mysql.NewDatabase(conf.SQL)

	// Init Router
	r := mux.NewRouter()

	// Route Handler / Endpoints
	r.HandleFunc("/add", Insert).Methods("GET")
	r.HandleFunc("/search/{key}", Retrieve).Methods("GET")
	r.HandleFunc("/", LiveCheck).Methods("GET")

	var err error
	err = http.ListenAndServe(":8080", r)

	return err
}
