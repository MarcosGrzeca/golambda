package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type Payload struct {
	Product int `json:"product"`
	SKU     int `json:"sku"`
}

type Body struct {
	URL     string  `json:"url"`
	Payload Payload `json:"payload"`
}

type Message struct {
	MessageId         string                                `json:"messageId"`
	Body              Body                                  `json:"body"`
	MessageAttributes map[string]events.SQSMessageAttribute `json:"messageAttributes"`
	Attributes        map[string]string                     `json:"attributes"`
	MessageGroupID    string
}

// HandleRequest processes SQS events and calls the external API.
func HandleRequest(ctx context.Context, event events.SQSEvent) (map[string]interface{}, error) {

	batchItemFailures := []map[string]interface{}{}
	var wg sync.WaitGroup
	mu := &sync.Mutex{}

	// Group messages by MessageGroupID
	messageGroups := make(map[string][]events.SQSMessage)
	for _, record := range event.Records {
		groupID := record.Attributes["MessageGroupID"]
		messageGroups[groupID] = append(messageGroups[groupID], record)
	}

	// Process each group sequentially
	for groupID, records := range messageGroups {
		wg.Add(1)

		go func(groupID string, records []events.SQSMessage) {
			defer wg.Done()

			log.Printf("Processing group ID: %s", groupID)
			for _, record := range records {
				log.Printf("Processing message ID: %s", record.MessageId)

				m, err := parse(record)
				if err != nil {
					fmt.Println("Error:", err)
					mu.Lock()

					batchItemFailures = append(batchItemFailures, map[string]interface{}{"itemIdentifier": record.MessageId})
					mu.Unlock()

					continue
				}

				statusCode, err := callExternalAPI(m.Body)
				if err != nil {
					mu.Lock()

					batchItemFailures = append(batchItemFailures, map[string]interface{}{"itemIdentifier": m.MessageId})
					mu.Unlock()

					log.Printf("Error processing message ID %s: %v", record.MessageId, err)
					continue
				}

				if statusCode >= 400 {
					mu.Lock()

					batchItemFailures = append(batchItemFailures, map[string]interface{}{"itemIdentifier": m.MessageId})
					mu.Unlock()

					log.Printf("API returned error for message ID %s: status code %d", record.MessageId, statusCode)
					continue
				}

				log.Printf("Successfully processed message ID: %s", record.MessageId)
			}
		}(groupID, records)
	}

	wg.Wait()

	sqsBatchResponse := map[string]interface{}{
		"batchItemFailures": batchItemFailures,
	}
	return sqsBatchResponse, nil
}

// callExternalAPI sends a POST request to an external API with the message body.
func callExternalAPI(body Body) (int, error) {
	requestBody, err := json.Marshal(
		body.Payload,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := http.Post(body.URL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, fmt.Errorf("failed to call external API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		responseBody, _ := ioutil.ReadAll(resp.Body)
		log.Printf("API error: %s", responseBody)
	}

	return resp.StatusCode, nil
}

func parse(event events.SQSMessage) (Message, error) {
	var msg Message
	msg.MessageAttributes = event.MessageAttributes
	msg.MessageId = event.MessageId
	msg.Attributes = event.Attributes

	fmt.Println(event.Attributes)

	if val, ok := event.Attributes["MessageGroupID"]; ok {
		msg.MessageGroupID = val
	}

	// Unmarshal the nested JSON in the Body field
	var body Body
	err := json.Unmarshal([]byte(event.Body), &body)
	if err != nil {
		fmt.Println("Error unmarshalling body JSON:", err)
		return Message{}, fmt.Errorf("error parsing JSON: %w", err) // Return empty Message and the error
	}
	msg.Body = body

	fmt.Printf("Message: %+v\n", msg)
	return msg, nil
}

func main() {
	log.Println("Beginning execution")
	lambda.Start(HandleRequest)
}
