package main

// Definition I need to know
// Mutex: Async mechanism that makes sur that only one thread operates on a single shared piece of data at a time.

// Primitive: A building block provided by a library to carry out and preform fundamental operations.


// Http has a ton of utilities for creating API functions
import (
	// "encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"mime/multipart"
	// "os"
	"io"
	
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
	
)


var (
    s3Session *session.Session
    s3Bucket  = "communal"
)

func init() {
    var err error
    s3Session, err = session.NewSession(&aws.Config{
        Region: aws.String("us-east-2"),
    })
    if err != nil {
        log.Fatalf("Failed to create session: %v", err)
    }
}




type KVStore struct {
	store map[string]string
	mu    sync.RWMutex
}
// KV Store Functions Start

func NewKVStore() *KVStore {
	return &KVStore{
		store: make(map[string]string),
	}
}

func (kvs *KVStore) Get(key string) (string, bool) {
	kvs.mu.RLock()
	defer kvs.mu.RUnlock()
	value, exists := kvs.store[key]
	return value, exists
}

func (kvs *KVStore) Set(key, value string) {
	kvs.mu.Lock()
	defer kvs.mu.Unlock()
	kvs.store[key] = value
}

func (kvs *KVStore) Delete(key string) {
	kvs.mu.Lock()
	defer kvs.mu.Unlock()
	delete(kvs.store, key)
}

//KVStore Functions End

//Photo Functions Start



func uploadToS3(file multipart.File, fileName string) (string, error) {
    uploader := s3manager.NewUploader(s3Session)

    result, err := uploader.Upload(&s3manager.UploadInput{
        Bucket: aws.String(s3Bucket),
        Key:    aws.String(fileName),
        Body:   file,
    })
    if err != nil {
        return "", err
    }

    return result.Location, nil
}


// This upload photo currently writes to the given OS, need to refactor in order to save photo to the 
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    file, header, err := r.FormFile("photo")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()

    location, err := uploadToS3(file, header.Filename)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte(location))
}

func retrieveFromS3(key string) (io.ReadCloser, error) {
    svc := s3.New(s3Session)

    result, err := svc.GetObject(&s3.GetObjectInput{
        Bucket: aws.String(s3Bucket),
        Key: aws.String(key),
    })
	
    if err != nil {
        return nil, err
    }
	
    return result.Body, nil
}

func retrieveHandler(w http.ResponseWriter, r *http.Request) {
    key := r.URL.Path[len("/photos/"):]

    body, err := retrieveFromS3(key)
    if err != nil {
		
        http.Error(w, "Photo not found", http.StatusNotFound)
        return
    }
    defer body.Close()

    w.Header().Set("Content-Type", "image/jpeg")
    io.Copy(w, body)
}



// Basic design pattern for API

// For each basic action, check to see which url its coming under.
// EX: /photo will handle all photo related api calls, and then
// will be seperatedd based on EVENT_TYPE


func main() {
	
	
	// s3 access with local key
	// key stored locally ~/.aws/credentials
	
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-2")},
		
	)
	
	svc := s3.New(sess)
	// kvs := NewKVStore()

	result, err := svc.ListBuckets(nil)
	if err != nil {
		fmt.Printf("Unable to list buckets, %v", err)
	}
	
	fmt.Println("Buckets:")
	
	for _, b := range result.Buckets {
		fmt.Printf("* %s created on %s\n",
			aws.StringValue(b.Name), aws.TimeValue(b.CreationDate))
	}
	    	
	http.HandleFunc("/photo" , func(w http.ResponseWriter, r *http.Request){
	    eventName := r.URL.Query().Get("event_name")
		
		if eventName == "" {
			http.Error(w, "No event name specified", http.StatusBadRequest)
			return
		}else if eventName == "UPLOAD_PHOTO"{
			if r.Method != "POST"{
				http.Error(w, "Invalid Request Method", http.StatusMethodNotAllowed)
				return
			}

			file, header, err := r.FormFile("photo")

			if err != nil {
				
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer file.Close()

			location,err := uploadToS3(file,header.Filename)
			if err !=nil {
				
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(location))
		}else if eventName == "FETCH_PHOTO"{
			if r.Method != "GET"{
				http.Error(w, "Invalid Request Method", http.StatusMethodNotAllowed)
				return
			}
			key := r.FormValue("photo")
			print(key)
			body, err := retrieveFromS3(key)
			if err != nil {
				http.Error(w, "Photo not found", http.StatusNotFound)
				return
			}
			defer body.Close()
		
			w.Header().Set("Content-Type", "image/jpeg")
			io.Copy(w, body)
			
		}
	})



	// // Basic KV Store
	// http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
	// 	key := r.URL.Query().Get("key")
	// 	if key == "" {
	// 		http.Error(w, "A key is required to use Get", http.StatusBadRequest)
	// 		return
	// 	}
	// 	value, exists := kvs.Get(key)
	// 	if !exists {
	// 		http.Error(w, "Key not found", http.StatusNotFound)
	// 		return
	// 	}
	// 	w.Write([]byte(value))
	// })

	// http.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
	// 	var data map[string]string
	// 	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
	// 		http.Error(w, "Invalid req payload", http.StatusBadRequest)
	// 	}
	// 	key, keyExists := data["key"]
	// 	value, valueExists := data["value"]
	// 	if !keyExists || !valueExists {
	// 		http.Error(w, "Key and value are both required for set", http.StatusBadRequest)
	// 	}
	// 	kvs.Set(key, value)
	// 	w.Write([]byte("OK"))
	// })

	// http.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
	// 	key := r.URL.Query().Get("key")
	// 	if key == "" {
	// 		http.Error(w, "Key is required to delete", http.StatusBadRequest)
	// 		return
	// 	}
	// 	kvs.Delete(key)
	// 	w.Write([]byte("OK"))
	// })

	// // End KVStore
	// http.HandleFunc("/uploadphoto", uploadPhoto)
	// http.HandleFunc("/fetchphotos/", fetchPhoto)
	



	fmt.Println("Starting Server on local:8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))

}
