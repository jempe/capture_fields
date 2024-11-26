# Capture Data Service

This Golang program provides a web service for capturing and validating user input data based on a pre-defined set of field rules. The captured data is stored in a BoltDB database, and can be exported as a CSV file.

## Dependencies

To run the program, you need to install the following dependencies:

- github.com/asaskevich/govalidator
- github.com/boltdb/bolt
- github.com/goji/httpauth
- github.com/spf13/viper

## Configuration

A configuration file config.toml should be located in the config folder. This file defines the port number for the web service, authentication credentials for accessing the exported data, and field validation rules.

Example of config.toml:

```
port = "8080"
auth.username = "admin"
auth.password = "password"

[fields]
  [fields.name]
  label = "Name"
  validation = "alpha"
  required = "true"

  [fields.email]
  label = "Email"
  validation = "email"
  required = "true"
```

## Endpoints

There are two endpoints provided by the web service:

/capture_data: Accepts POST requests with form data for capturing and validation based on the field rules defined in the configuration file.

/data.csv: Returns a CSV file with the captured data, requires HTTP Basic authentication with the credentials defined in the configuration file.


## Usage

1. Set up the configuration file with the desired field validation rules and authentication credentials.
2. Run the program with go run main.go.
3. Send a POST request with form data to /capture_data to capture and validate the data.
4. Access /data.csv with the authentication credentials to download the captured data as a CSV file.

## Functions

The program contains the following functions:

main(): Initializes the database, reads the configuration file, and sets up the HTTP server.

capture_handler(w http.ResponseWriter, r *http.Request): Handles the /capture_data endpoint for capturing and validating form data.

csv_handler(w http.ResponseWriter, r *http.Request): Handles the /data.csv endpoint for exporting captured data as a CSV file.

initDb(): Initializes the BoltDB database.

checkErr(err error): Checks for errors and panics if any are found.

RandomBytes(length int): Generates random bytes of the specified length.

RandomString(length int): Generates a random string of the specified length.
