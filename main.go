package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	transport "github.com/aws/smithy-go/endpoints"
)

var _ s3.EndpointResolverV2 = &S3Endpoint{}

func GetEnvOrPanic(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	panic("Environment variable " + key + " not found")
}

func GetEnvOrDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		log.Println("Using environment variable for", key, ":", value)
		return value
	}
	log.Println("Using default value for", key, ":", defaultValue)
	return defaultValue
}

var (
	BUCKET      = GetEnvOrPanic("BUCKET")
	REGION      = GetEnvOrDefault("REGION", "us-east-1")
	S3_ENDPOINT = GetEnvOrPanic("S3_ENDPOINT")

	AWS_ACCESS_KEY = GetEnvOrPanic("AWS_ACCESS_KEY")
	AWS_SECRET_KEY = GetEnvOrPanic("AWS_SECRET_KEY")
)

type S3Endpoint struct{}

// ResolveEndpoint implements s3.EndpointResolverV2.
func (s *S3Endpoint) ResolveEndpoint(ctx context.Context, params s3.EndpointParameters) (transport.Endpoint, error) {
	path := ""
	if params.Bucket != nil {
		path = *params.Bucket
	}

	return transport.Endpoint{
		URI: url.URL{Scheme: "https", Host: S3_ENDPOINT, Path: path},
	}, nil
}

func main() {
	// Init S3 Client
	ctx := context.Background()
	rwLock := sync.RWMutex{}

	c := s3.New(s3.Options{
		EndpointResolverV2: &S3Endpoint{},
		Credentials:        credentials.NewStaticCredentialsProvider(AWS_ACCESS_KEY, AWS_SECRET_KEY, ""),
		Region:             REGION,
	})

	objChan := make(chan types.Object, 1000)

	wg := sync.WaitGroup{}

	wg.Add(1)

	go func() {
		defer wg.Done()
		var cursor *string = nil
		count := 0
		for {
			objects, err := c.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
				Bucket:            aws.String(BUCKET),
				ContinuationToken: cursor,
			})
			if err != nil {
				panic(err)
			}
			count += len(objects.Contents)
			log.Println("objects", count)
			for _, obj := range objects.Contents {
				objChan <- obj
			}
			cursor = objects.NextContinuationToken
			if cursor == nil {
				break
			}
		}
		close(objChan)
	}()

	objects := map[string]types.Object{}
	directory := map[string][]string{}

	go func() {
		for obj := range objChan {
			key := *obj.Key

			// fetch extension
			ext := filepath.Ext(key)
			dir := filepath.Base(filepath.Dir(key))
			if ext == ".jpg" || ext == ".jpeg" {
				rwLock.Lock()
				objects[key] = obj
				directory[dir] = append(directory[dir], key)
				rwLock.Unlock()
			}
		}
		log.Println("Done fetching objects")
		log.Println("Total objects:", len(objects))
	}()

	type Metadata struct {
		Key string `json:"key"`
		Dir string `json:"dir"`
	}

	// run http server that shows one of the objects randomly
	server := http.NewServeMux()
	server.HandleFunc("/random", func(w http.ResponseWriter, r *http.Request) {
		// send headers
		w.Header().Set("Content-Type", "application/json")

		randomeList := []Metadata{}
		for _, obj := range objects {
			if len(randomeList) >= 100 {
				break
			}
			randomeList = append(randomeList, Metadata{
				Key: *obj.Key,
				Dir: filepath.Base(filepath.Dir(*obj.Key)),
			})
		}
		json.NewEncoder(w).Encode(randomeList)
	})

	server.HandleFunc("/dir/", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Path[len("/dir/"):]
		json.NewEncoder(w).Encode(directory[dir])
	})

	// return object when key is provided
	server.HandleFunc("/obj/", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path[len("/obj/"):]

		presignClient := s3.NewPresignClient(c)
		presignReq, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(BUCKET),
			Key:    aws.String(key),
		}, s3.WithPresignExpires(time.Hour))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		http.Redirect(w, r, presignReq.URL, http.StatusTemporaryRedirect)
	})

	// static files
	server.Handle("/_app/", http.FileServer(http.Dir("./build")))
	server.Handle("/favicon.ico", http.NotFoundHandler())
	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./build/index.html")
	})

	// start server
	log.Println("Server started at http://localhost:8080")
	http.ListenAndServe(":8080", &HttpLogger{Handler: server})
}

type HttpLogger struct {
	http.Handler
}

func (h *HttpLogger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
	h.Handler.ServeHTTP(w, r)
}
