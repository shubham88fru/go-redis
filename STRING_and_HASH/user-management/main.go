package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

const userHashNamePrefix = "user:"
const useIdCounter = "userid_counter"

var client *redis.Client

/* 
	Init is a special function in Go that is 
	called before the main function.
	Here, we are initializing the Redis client and 
	checking if we can connect to the Redis server.
	If we can't connect to the Redis server, 
	we will log a fatal error and stop the execution of the program.
*/
func init() {
	client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	err := client.Ping(context.Background()).Err()

	if err != nil {
		log.Fatalf("failed to connect to redis. error message - %v", err)
	}

	log.Println("Succesfully connected to Redis..")
}


func main() {
	r := mux.NewRouter()

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods(http.MethodGet)

	r.HandleFunc("/", add).Methods(http.MethodPost)
	r.HandleFunc("/{id}", get).Methods(http.MethodGet)

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}


//create a user.
func add(w http.ResponseWriter, r *http.Request) {
	var user map[string]string
	err := json.NewDecoder(r.Body).Decode(&user)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request payload"))
		return
	}

	log.Println("User data received - ", user)

	id, err := client.Incr(context.Background(), useIdCounter).Result()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to generate user id"))
		return
	}

	userHashName := userHashNamePrefix + strconv.Itoa(int(id))

	err = client.HSet(context.Background(), userHashName, user).Err()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create user"))
		return
	}

	w.Header().Add("Location", "http://"+r.Host+"/"+strconv.Itoa(int(id)))
	w.WriteHeader(http.StatusCreated)

	log.Println("User created with id - ", id)
}


func get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	log.Println("Search for user", id)

	userHashName := userHashNamePrefix + id
	user, err := client.HGetAll(context.Background(), userHashName).Result()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get user"))
		return
	}

	if len(user) == 0 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("User not found"))
		return
	}

	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to get user"))
		return
	}
}