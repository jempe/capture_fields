package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"os"
	"regexp"
	"time"

	"github.com/boltdb/bolt"
)

var Db *bolt.DB

type CaptureResponse struct {
	Success     bool     `json:"success"`
	ErrorFields []string `json:"error_fields"`
}

type Config struct {
	Port   int              `json:"port"`
	Auth   AuthConfig       `json:"auth"`
	ID     string           `json:"id"`
	Fields map[string]Field `json:"fields"`
}

type AuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Field struct {
	Label      string `json:"label"`
	Validation string `json:"validation,omitempty"`
	Regex      string `json:"regex,omitempty"`
	Required   string `json:"required,omitempty"`
}

var config Config

func main() {
	err := initDb()
	checkErr(err)

	err = readConfig("config/config.json", &config) // Read the config file
	checkErr(err)

	port := fmt.Sprint(config.Port)

	log.Println("Capture fields web service started on http://127.0.0.1:" + port)

	http.HandleFunc("/capture_data", capture_handler)
	http.Handle("/data.csv", basicAuth(csv_handler, config.Auth.Username, config.Auth.Password))
	panic(http.ListenAndServe(":"+port, nil))
}

func readConfig(filename string, config *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(config)
	if err != nil {
		return err
	}

	return nil
}

func capture_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	var response CaptureResponse

	fields := map[string]map[string]string{
		"name": {
			"label":      "Name",
			"validation": "alpha",
			"required":   "true",
		},
		"email": {
			"label":      "Email",
			"validation": "email",
			"required":   "true",
		},
		// Add more fields here
	}

	r.ParseForm()

	store_data := make(map[string]string)

	var field_errors []string
	var row_id string

	for field_name, field_data := range fields {
		var field_value string
		field_input := r.Form[field_name]

		if len(field_input) > 0 {
			field_value = field_input[0]
		}

		store_data[field_name] = field_value

		validation := field_data["validation"]
		required := field_data["required"]

		if required == "true" && field_value == "" {
			field_errors = append(field_errors, field_name)
		} else if field_value != "" {
			if validation == "email" {
				_, err := mail.ParseAddress(field_value)
				if err != nil {
					field_errors = append(field_errors, field_name)
				}
			} else if validation == "numeric" {
				numericPattern := regexp.MustCompile(`^[0-9]+$`)
				if !numericPattern.MatchString(field_value) {
					field_errors = append(field_errors, field_name)
				}
			} else if validation == "url" {
				urlPattern := regexp.MustCompile(`^https?:\/\/[^\s]+$`)
				if !urlPattern.MatchString(field_value) {
					field_errors = append(field_errors, field_name)
				}
			} else if validation == "alpha" {
				alphaPattern := regexp.MustCompile(`^[a-zA-Z]+$`)
				if !alphaPattern.MatchString(field_value) {
					field_errors = append(field_errors, field_name)
				}
			} else if validation == "alphanumeric" {

				alphaNumericPattern := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
				if !alphaNumericPattern.MatchString(field_value) {
					field_errors = append(field_errors, field_name)
				}
			} else if validation == "regex" {
				regex := field_data["regex"]
				re := regexp.MustCompile(regex)

				if !re.MatchString(field_value) {
					field_errors = append(field_errors, field_name)
				}
			} else if validation == "none" {

			} else {
				log.Println("invalid validation type:", validation)
			}
		}
	}

	if row_id == "" {
		row_id, _ = RandomString(32)
	}

	response.ErrorFields = field_errors

	if len(field_errors) == 0 {
		_ = Db.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("data"))

			t := time.Now()

			store_data["timestamp"] = t.Format("1/2/2006 15:04")

			store_data_json, err := json.Marshal(store_data)

			if err == nil {
				b.Put([]byte(row_id), store_data_json)
			}

			return nil
		})

		response.Success = true
	}

	json_response, _ := json.Marshal(response)

	fmt.Fprintln(w, string(json_response))
}

func csv_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")

	b := &bytes.Buffer{}
	wr := csv.NewWriter(b)

	fieldList := config.Fields

	var column_names []string
	var column_ids []string

	for fieldName, field := range fieldList {

		column_ids = append(column_ids, fieldName)
		column_names = append(column_names, field.Label)

	}

	column_ids = append(column_ids, "timestamp")
	column_names = append(column_names, "Timestamp")

	wr.Write(column_names)

	_ = Db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte("data"))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var row map[string]string
			err := json.Unmarshal(v, &row)

			if err == nil {
				var record []string

				for _, column_id := range column_ids {
					var column_value string

					if val, ok := row[column_id]; ok {
						column_value = val
					}

					record = append(record, column_value)
				}

				wr.Write(record)

			}
		}

		return nil
	})

	wr.Flush()

	fmt.Fprintln(w, b)
}

func basicAuth(handler http.HandlerFunc, username, password string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()

		if !ok || user != username || pass != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}

		handler(w, r)
	}
}

func initDb() (err error) {
	Db, err = bolt.Open("config/captured_data.db", 0644, nil)
	if err != nil {
		return err
	}

	err = Db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("data"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})

	return err
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

// RandomBytes is useful to generate HMAC key
func RandomBytes(length int) (key []byte, err error) {
	randomBytes := make([]byte, length)

	_, err = rand.Read(randomBytes)
	if err == nil {
		key = randomBytes
	}

	return key, err
}

// RandomBytes is useful to generate HMAC key
func RandomString(length int) (key string, err error) {
	randomBytes, err := RandomBytes(length)

	if err == nil {
		key = base64.URLEncoding.EncodeToString(randomBytes)
	}

	return key, err
}
