package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestHandleRequest(t *testing.T) {
	// Create a sample SQS event
	sqsEvent := events.SQSEvent{
		Records: []events.SQSMessage{
			{
				MessageId: "1",
				Body:      `{"url": "https://example.com", "payload": {"product": 10, "sku": 20}}`,
				Attributes: map[string]string{
					"MessageGroupID": "group1",
				},
			},
			{
				MessageId: "2",
				Body:      `{"url": "https://example.com", "payload": {"product": 30, "sku": 40}}`,
				Attributes: map[string]string{
					"MessageGroupID": "group2",
				},
			},
			{
				MessageId:  "3",
				Body:       `{"url": "https://example.com", "payload": {"product": 50, "sku": 60}}`,
				Attributes: map[string]string{
					// No MessageGroupID to test random group ID generation
				},
			},
		},
	}

	// Call the HandleRequest function
	response, err := HandleRequest(context.Background(), sqsEvent)

	// Assert no error
	assert.NoError(t, err)

	// Assert the response
	expectedResponse := map[string]interface{}{
		"batchItemFailures": []map[string]interface{}{},
	}
	assert.Equal(t, expectedResponse, response)
}

func TestParse(t *testing.T) {
	// Create a sample SQS message
	sqsMessage := events.SQSMessage{
		MessageId: "1",
		Body:      `{"url": "https://example.com", "payload": {"product": 10, "sku": 20}}`,
		Attributes: map[string]string{
			"MessageGroupID": "group1",
		},
	}

	// Call the parse function
	message, err := parse(sqsMessage)

	// Assert no error
	assert.NoError(t, err)

	// Assert the parsed message
	expectedMessage := Message{
		MessageID: "1",
		Body: Body{
			URL: "https://example.com",
			Payload: Payload{
				Product: 10,
				SKU:     20,
			},
		},
		Attributes: map[string]string{
			"MessageGroupID": "group1",
		},
		MessageGroupID: "group1",
	}
	assert.Equal(t, expectedMessage, message)
}

func TestCallExternalAPI(t *testing.T) {
	// Mock the external API response
	httpClient := &http.Client{
		Transport: &mockTransport{
			response: &http.Response{
				StatusCode: 200,
				Body:       ioutil.NopCloser(bytes.NewBufferString(`OK`)),
			},
		},
	}

	// Replace the default HTTP client with the mock client
	http.DefaultClient = httpClient

	// Create a sample body
	body := Body{
		URL: "https://example.com",
		Payload: Payload{
			Product: 10,
			SKU:     20,
		},
	}

	// Call the callExternalAPI function
	statusCode, err := callExternalAPI(body)

	// Assert no error
	assert.NoError(t, err)

	// Assert the status code
	assert.Equal(t, 200, statusCode)
}

// mockTransport is a mock implementation of http.RoundTripper
type mockTransport struct {
	response *http.Response
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.response, nil
}
