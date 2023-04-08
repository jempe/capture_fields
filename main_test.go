package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var ts *httptest.Server

// TestCaptureHandler_Success tests the capture_handler with valid form data.
func TestCaptureHandler(t *testing.T) {
	err := initDb()
	assert.NoError(t, err)

	viper.SetConfigName("config")
	viper.AddConfigPath("config/")

	err = viper.ReadInConfig() // Find and read the config file
	assert.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(capture_handler))

	formData := "name=John&email=johndoe@example.com"
	resp, err := http.Post(ts.URL, "application/x-www-form-urlencoded", strings.NewReader(formData))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	var captureResponse CaptureResponse
	err = json.Unmarshal(body, &captureResponse)
	assert.NoError(t, err)

	assert.True(t, captureResponse.Success)
	assert.Empty(t, captureResponse.ErrorFields)

	resp.Body.Close()

	formData = "name=John&email=johndfsd"
	resp, err = http.Post(ts.URL, "application/x-www-form-urlencoded", strings.NewReader(formData))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err = ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	err = json.Unmarshal(body, &captureResponse)
	assert.NoError(t, err)

	assert.False(t, captureResponse.Success)
	assert.NotEmpty(t, captureResponse.ErrorFields)

	resp.Body.Close()

	ts.Close()
}

// TestRandomBytes tests the generation of random bytes with the given length.
func TestRandomBytes(t *testing.T) {
	length := 32
	randomBytes, err := RandomBytes(length)
	assert.NoError(t, err)
	assert.Len(t, randomBytes, length)
}
