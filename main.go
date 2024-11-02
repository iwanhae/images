package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	transport "github.com/aws/smithy-go/endpoints"
)

var _ s3.EndpointResolverV2 = &S3Endpoint{}

type S3Endpoint struct{}

// ResolveEndpoint implements s3.EndpointResolverV2.
func (s *S3Endpoint) ResolveEndpoint(ctx context.Context, params s3.EndpointParameters) (transport.Endpoint, error) {
	path := ""
	if params.Bucket != nil {
		path = *params.Bucket
	}

	return transport.Endpoint{
		URI: url.URL{Scheme: "https", Host: "s3-glacier.iwanhae.kr", Path: path},
	}, nil
}

func main() {
	// Init S3 Client
	ctx := context.Background()
	rwLock := sync.RWMutex{}

	c := s3.New(s3.Options{
		EndpointResolverV2: &S3Endpoint{},
		Credentials:        credentials.NewStaticCredentialsProvider("iwanhae", "doraemon96", ""),
		Region:             "us-east-1",
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
				Bucket:            aws.String("xx-knit-bid"),
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

	go func() {
		for obj := range objChan {
			key := *obj.Key

			// fetch extension
			ext := filepath.Ext(key)
			if ext == ".jpg" || ext == ".jpeg" {
				rwLock.Lock()
				objects[key] = obj
				rwLock.Unlock()
			}
		}
		log.Println("Done fetching objects")
		log.Println("Total objects:", len(objects))
	}()

	imageCache := make(chan []byte, 50)

	go func() {
		for {
			// pick one object randomly
			var obj types.Object
			for k := range objects {
				rwLock.RLock()
				obj = objects[k]
				rwLock.RUnlock()
				break
			}
			if obj.Key == nil {
				continue
			}

			// get stream from s3
			stream, err := c.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String("xx-knit-bid"),
				Key:    obj.Key,
			})
			if err != nil {
				panic(err)
			}
			log.Println("Fetched", *obj.Key)
			image, err := io.ReadAll(stream.Body)
			if err != nil {
				panic(err)
			}
			imageCache <- image
		}
	}()

	// run http server that shows one of the objects randomly
	server := http.NewServeMux()
	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		image := <-imageCache

		// send headers
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", strconv.Itoa(len(image)))

		// stream the object
		io.Copy(w, bytes.NewReader(image))
	})
	// handle favicon
	server.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	// start server
	log.Println("Server started at http://localhost:8080")
	http.ListenAndServe(":8080", server)
}
