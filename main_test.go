package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCaptureHandler_Success tests the capture_handler with valid form data.
func TestCaptureHandler_Success(t *testing.T) {
	err := initDb()
	assert.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(capture_handler))
	defer ts.Close()

	formData := "name=John&email=johndoe@example.com"
	resp, err := http.Post(ts.URL, "application/x-www-form-urlencoded", strings.NewReader(formData))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	defer resp.Body.Close()

	var captureResponse CaptureResponse
	err = json.Unmarshal(body, &captureResponse)
	assert.NoError(t, err)

	assert.True(t, captureResponse.Success)
	assert.Empty(t, captureResponse.ErrorFields)
}

// TestCaptureHandler_InvalidEmail tests the capture_handler with an invalid email.
func TestCaptureHandler_InvalidEmail(t *testing.T) {
	err := initDb()
	assert.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(capture_handler))
	defer ts.Close()

	formData := "name=John&email=invalid-email"
	resp, err := http.Post(ts.URL, "application/x-www-form-urlencoded", strings.NewReader(formData))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	defer resp.Body.Close()

	var captureResponse CaptureResponse
	err = json.Unmarshal(body, &captureResponse)
	assert.NoError(t, err)

	assert.False(t, captureResponse.Success)
	assert.Contains(t, captureResponse.ErrorFields, "email")
}

// TestRandomBytes tests the generation of random bytes with the given length.
func TestRandomBytes(t *testing.T) {
	length := 32
	randomBytes, err := RandomBytes(length)
	assert.NoError(t, err)
	assert.Len(t, randomBytes, length)
}

// TestRandomString tests the generation of random strings with the given length.
func TestRandomString(t *testing.T) {
	length := 32
	randomString, err := RandomString(length)
	assert.NoError(t, err)
	assert.Len(t, randomString, length)
}
