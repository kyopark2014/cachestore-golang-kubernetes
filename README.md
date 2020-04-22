# cachestore-golang-kubernetes

It is an implementation of cached store based on golang in kubernetes where the combination of redis cache and MySQL is applied.
The caching mechanism can enhance the performance of a system in which a temporary memory was used between the application and the persistent database. So, cache memory stores recently used data items in order to reduce the number of database hits as much as possible.



<img src="https://user-images.githubusercontent.com/52392004/79468854-6b9aff80-803a-11ea-872d-01aa7720d32d.png" width="70%"></img>
1) query the DNS address of the destination where the ip address of Load Balancer is used 
2) All requests from clients will be through to load balance which provides scale-out
3) One of applications servers will be allocated by round-robin scheduling
4) Caching is an effective way to reduce the number of access toward database which is usually the bottleneck when the scale was extended
5) Cached data will be expired and then needs to query from database
6) The amount of queries for database can be reduced once a caching algorithm was used


### Initiation

Initialize a radis to cache data

```go
func Initialize() error {
    ....
    // initialize radis for in-memory cache
    rediscache.NewRedisCache(conf.Redis)
}
```

MySQL also initialized as bellow

```go
db.MyDb, sqlerr = sql.Open("mysql", DSN)
if sqlerr != nil {
	return sqlerr
}
defer db.MyDb.Close()
```

Before define apis, the database and table are initiated as bellow.

```go
// create database
create, err := MyDb.Query("CREATE DATABASE IF NOT EXISTS " + Dbname)
if err != nil {
	log.E("Fail to create database %v", err)
	os.Exit(1)
}
defer create.Close()
```

```go
// Create table
var statement string = "CREATE TABLE IF NOT EXISTS " + Dbname + "." + Dbtable + " (" + "uid VARCHAR(20), name VARCHAR(20), email VARCHAR(30), age BIGINT" + ")"
createTable, err := MyDb.Query(statement)
if err != nil {
	log.E("Fail to create database %v", err)
	os.Exit(1)
}
defer createTable.Close()
```

### REST API
Now I am using mux to deploy RESTful API.

```go
r := mux.NewRouter()

// Route Handler / Endpoints
r.HandleFunc("/add", Insert).Methods("GET")
r.HandleFunc("/search/{key}", Retrieve).Methods("GET")

var err error
err = http.ListenAndServe(":8000", r)
```

Lets define Insert() and Retrieve().

```go
func Insert(w http.ResponseWriter, r *http.Request) {
	var value data.UserProfile
	_ = json.NewDecoder(r.Body).Decode(&value)

	err := db.InsertToDB(value)
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
```

```go
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
		value, errCode := db.RetrevefromDB(uid)
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
```


## MySQL


#### Go-MySQL-Driver

Go-MySQL-Driver is used for SQL.

https://github.com/go-sql-driver/mysql 

##### install 
```c
$ go get -u github.com/go-sql-driver/mysql
```

##### import sql driver in main.go

```go
import( _ "github.com/go-sql-driver/mysql")
```

##### data type
This is a profile to represent a person for an example.
```c
type UserProfile struct {
	UID   string `json:"uid"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}
```


Let me define InsertToDB to put input data into database
```go
func InsertToDB(value data.UserProfile) error {
	insert, err := MyDb.Query("INSERT INTO "+Dbname+"."+Dbtable+" (uid, name, email, age) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE uid=?",
		value.UID, value.Name, value.Email, strconv.FormatInt(int64(value.Age), 10), value.UID)
	if err != nil {
		log.E("Fail to insert data %v", err)
		return err
	}
	defer insert.Close()
	log.D("Successfully inserted in SQL database")

	return nil
}
```

Define RetrevefromDB to get a cached data from redis. If there is no data in redis cache, it will check the database.

```go
func RetrevefromDB(uid string) (data.UserProfile, int) {
	var value data.UserProfile

	results, err := MyDb.Query("SELECT * FROM "+Dbname+"."+Dbtable+" WHERE uid = ?", uid)
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
```




## Initiate and StartService

The main is initiating and starting the service.

```go
err := Initialize()
if err != nil {
	log.E("Failed to initialize service: %v", err)
	os.Exit(1)
}

err = StartService()
if err != nil {
	log.E("Failed to start service: %v", err)
	os.Exit(1)
}
log.E("Exiting service ...")
```

The main function for the restful api is starting from server.InitServer() in "main.go".
  
```go
func StartService() error {
	log.D("start the service...")

	var err error
	if err = server.InitServer(); err != nil {
	log.E("Failed to start the HTTP(s) server: err:[%v]", err)
}
```  




## Configuration

#### config.go

Define AppConfig in order to load the configuration

```go
var config *AppConfig
  
func GetInstance() *AppConfig {
	if config == nil {
		config = &AppConfig{}
	}
	return config
}
  
Logging struct {
	Enable bool   `json:"Enable"`
	Level  string `json:"Level"`
} 
```




## Logging

#### main.go

The log level is setting using SetupLogger.

```go
import ("cachestore-golang-kubernetes/pkg/log")

log.SetupLogger(conf.Logging.Enable, conf.Logging.Level)
```

Then, it is used as bellow.

```go
log.I("Starting service ...")
log.E("Failed to load config file: %s", configFileName)
```

#### log.go

The log level is appliying as bellow.

```go
func SetupLogger(isEnabled bool, level string) {
	loggingEnable = isEnabled
	backend := logging.NewLogBackend(os.Stdout, "", 0)

	backendFormatter := logging.NewBackendFormatter(backend, format)

	var lvl logging.Level
	switch level {
	case "ERROR":
		lvl = logging.ERROR
	case "WARNING":
		lvl = logging.WARNING
	case "INFO":
		lvl = logging.INFO
	case "DEBUG":
		lvl = logging.DEBUG
	default:
		lvl = logging.INFO
	}

	backendLeveled := logging.AddModuleLevel(backendFormatter)
	backendLeveled.SetLevel(lvl, "")

	logging.SetBackend(backendLeveled)
}

// D writes debug level log
func D(format string, v ...interface{}) {
	if loggingEnable {
		log.Debugf(format, v...)
	}
}

// E writes error level log
func E(format string, v ...interface{}) {
	if loggingEnable {
		log.Errorf(format, v...)
	}
}
```




## Kubernetes
In oreder to explain easily, minikube will be used.

### Create Minikube
```c
$ sudo minikube start --vm-driver=none --disk-size 40g
```
Note that no vm-driver is helpful for the performant to test in local environment but it requires recent minikube. I recommend to use v1.9.2.


### Deploy and initialize MySQL
```c
$ helm install --wait \
      --set mysqlPassword=minikube \
      --set mysqlUser=minikube \
      --set mysqlRootPassword=minikube \
      --set persistence.size=4Gi \
      --name mydb \
      stable/mysql

$ mysql -h 10.105.102.33 -uroot -pminikube -e "GRANT ALL PRIVILEGES ON *.* TO 'minikube' WITH GRANT OPTION" 
```

### Deploy Redis
```c
$ helm install --wait --set usePassword=false --name redis stable/redis
```

### Create Secret
```c
$ kubectl create secret docker-registry artifactory-dockerhub \
    --docker-server="https://index.docker.io/v1/" \
    --docker-username="id" \
    --docker-password="password" \
    --docker-email=myid@mail.com \
    -o yaml --dry-run | kubectl replace -n common --force -f -
```

### Set the information of Repository in values.yaml
```c
image:
  repository: docker.io/id/myapps
  tag: v1
  pullPolicy: IfNotPresent

imagePullSecrets: 
  - name: artifactory-dockerhub
```

### Install
```c
$ helm install cachestore
```

## The result
#### Set data

The input is using HTTP POST api as bellow.

POST: /add 

```go
{
    "uid": "kyopark",
    "name": "John",
    "email": "john@email.com",
    "age": 24
}
```


GET: /search/kyopark  
* where "kyopark" is one of idenfication for a profile

- Before expiration of redis cache

```c
2020-04-16 10:36:22.871 [D] value from cache: &{UID:kyopark Name:John Email:john@email.com Age:24}
```

- After expiration of redis cache
```c
2020-04-16 10:37:44.977 [D] Successfully quaried in database: {UID:kyopark Name:John Email:john@email.com Age:24}
```



## Reference

https://tutorialedge.net/golang/golang-mysql-tutorial/

https://www.youtube.com/watch?v=DWNozbk_fuk
  
https://www.youtube.com/watch?v=SonwZ6MF5BE
