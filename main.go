package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"syscall"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/gorilla/mux"
)

const freqCount = "fileRequestsCount"

func handler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	name := query.Get("name")
	if name == "" {
		name = "Guest"
	}
	log.Printf("Received request for %s\n", name)
	w.Write([]byte(fmt.Sprintf("Hello, %s\n", name)))
}

var fpath = "/tmp/test.txt"

func main() {
	//os.Setenv("REDIS_SERVICE", "localhost")
	//os.Setenv("REDIS_PORT", "6379")

	// Create Server and Route Handlers
	r := mux.NewRouter()

	r.HandleFunc("/", handler)
	rclient := newRedisClient()
	rclient.Set(freqCount, 0, time.Second*1000000)
	r.HandleFunc("/file", fileHandler)
	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	//createFile()
	//writeFile("Hello K8s!!\n")

	// Start Server
	go func() {
		log.Println("Starting Server")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// Graceful Shutdown
	waitForShutdown(srv)
}

func newRedisClient() *redis.Client {
	redisSvc := os.Getenv("REDIS_SERVICE")
	redisPort := os.Getenv("REDIS_PORT")
	redisUrl := redisSvc + ":" + redisPort
	//fmt.Printf("\nRedis url is:%v", redisUrl)

	client := redis.NewClient(&redis.Options{
		Addr:     redisUrl,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	pong, err := client.Ping().Result()
	fmt.Println(pong, err)
	return client
}

func waitForShutdown(srv *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-interruptChan

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)

	log.Println("Shutting down")
	os.Exit(0)
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	rclient := newRedisClient()
	reqCount, err := rclient.Get(freqCount).Result()
	if err == redis.Nil {
		fmt.Println("reqCount does not exist")
	} else if err != nil {
		fmt.Println(err)
	}
	count, _ := strconv.Atoi(reqCount)
	count++
	rclient.Set(freqCount, count, time.Second*1000000)
	//fmt.Printf("\nsTotal number of request:%v", count)
	writeFile(fmt.Sprintf("\nsTotal number of request:%v", count))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fp := path.Join(fpath)
	http.ServeFile(w, r, fp)
}

func createFile() {
	// detect if file exists
	var _, err = os.Stat(fpath)

	// create file if not exists
	if os.IsNotExist(err) {
		var file, err = os.Create(fpath)
		if isError(err) {
			return
		}
		defer file.Close()
	}

	fmt.Println("==> done creating file", fpath)
}

func writeFile(content string) {
	// open file using READ & WRITE permission
	var file, err = os.OpenFile(fpath, os.O_RDWR, 0644)
	if isError(err) {
		return
	}
	defer file.Close()

	// write some text line-by-line to file
	_, err = file.WriteString(content)
	if isError(err) {
		return
	}
	// save changes
	err = file.Sync()
	if isError(err) {
		return
	}

	fmt.Println("==> done writing to file")
}

func isError(err error) bool {
	if err != nil {
		fmt.Println(err.Error())
	}

	return (err != nil)
}
