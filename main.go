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
	"reflect"
	"regexp"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/boltdb/bolt"
	"github.com/goji/httpauth"
	"github.com/spf13/viper"
)

var Db *bolt.DB

type CaptureResponse struct {
	Success     bool     `json:"success"`
	ErrorFields []string `json:"error_fields"`
}

func main() {
	err := initDb()
	checkErr(err)

	viper.SetConfigName("config")
	viper.AddConfigPath("config/")

	err = viper.ReadInConfig() // Find and read the config file
	checkErr(err)

	log.Println("Capture fields web service started on http://127.0.0.1:" + viper.GetString("port"))

	http.HandleFunc("/capture_data", capture_handler)
	http.Handle("/data.csv", httpauth.SimpleBasicAuth(viper.GetString("auth.username"), viper.GetString("auth.password"))(http.HandlerFunc(csv_handler)))
	panic(http.ListenAndServe(":"+viper.GetString("port"), nil))
}

// validation values: email, numeric, url, alpha, alphanumeric, regex, "none"

func capture_handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	var response CaptureResponse

	fields := viper.GetStringMap("fields")

	r.ParseForm()

	store_data := make(map[string]string)

	var field_errors []string
	var row_id string

	for field_name, _ := range fields {
		var field_value string
		field_data := r.Form[field_name]

		if len(field_data) > 0 {
			field_value = field_data[0]
		}

		if field_name == viper.GetString("id") {
			row_id = field_value
		}

		store_data[field_name] = field_value

		field := reflect.ValueOf(fields[field_name])

		var validation string
		var regex string
		var required string

		for _, key := range field.MapKeys() {
			v := field.MapIndex(key)

			if key.String() == "validation" {
				validation = fmt.Sprintf("%v", v)
			} else if key.String() == "regex" {
				regex = fmt.Sprintf("%v", v)
			} else if key.String() == "required" {
				required = fmt.Sprintf("%v", v)
			}
		}

		if required == "true" && field_value == "" {
			field_errors = append(field_errors, field_name)
		} else if field_value != "" {
			if validation == "email" {
				if !govalidator.IsEmail(field_value) {
					field_errors = append(field_errors, field_name)
				}
			} else if validation == "numeric" {
				if !govalidator.IsNumeric(field_value) {
					field_errors = append(field_errors, field_name)
				}
			} else if validation == "url" {
				if !govalidator.IsURL(field_value) {
					field_errors = append(field_errors, field_name)
				}
			} else if validation == "alpha" {
				if !govalidator.IsAlpha(field_value) {
					field_errors = append(field_errors, field_name)
				}
			} else if validation == "alphanumeric" {
				if !govalidator.IsAlphanumeric(field_value) {
					field_errors = append(field_errors, field_name)
				}
			} else if validation == "regex" {
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

	if viper.GetString("id") == "" {
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

	fields := viper.GetStringMap("fields")

	var column_names []string
	var column_ids []string

	for field_name, _ := range fields {
		field := reflect.ValueOf(fields[field_name])

		for _, key := range field.MapKeys() {
			v := field.MapIndex(key)

			if key.String() == "label" {
				label := fmt.Sprintf("%v", v)
				column_ids = append(column_ids, field_name)
				column_names = append(column_names, label)
			}
		}

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
