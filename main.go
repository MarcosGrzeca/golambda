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
	"github.com/google/uuid"
)

type Body struct {
	URL     string          `json:"url"`
	Payload json.RawMessage `json:"payload"`
}

type Message struct {
	MessageId         string                                `json:"messageId"`
	Body              Body                                  `json:"body"`
	MessageAttributes map[string]events.SQSMessageAttribute `json:"messageAttributes"`
	Attributes        map[string]string                     `json:"attributes"`
	MessageGroupID    string
}

func groupMessagesByMessaGROUPID(event events.SQSEvent) map[string][]events.SQSMessage {

	// Group messages by MessageGroupID
	messageGroups := make(map[string][]events.SQSMessage)
	for _, record := range event.Records {
		//Standard queues do not have MessageGroupID
		groupID, ok := record.Attributes["MessageGroupID"]
		if !ok {
			groupID = uuid.New().String()
		}
		messageGroups[groupID] = append(messageGroups[groupID], record)

	}
	return messageGroups
}

// HandleRequest processes SQS events and calls the external API.
func HandleRequest(ctx context.Context, event events.SQSEvent) (map[string]interface{}, error) {

	batchItemFailures := []map[string]interface{}{}
	var wg sync.WaitGroup
	mu := &sync.Mutex{}

	messageGroups := groupMessagesByMessaGROUPID(event)

	// Process each group sequentially
	for groupID, records := range messageGroups {
		wg.Add(1)

		go func(groupID string, records []events.SQSMessage) {
			defer wg.Done()

			log.Printf("Processing group ID: %s", groupID)
			for _, record := range records {
				traceId := uuid.New().String()
				log.Printf("[%s]Processing message ID: %s", traceId, record.MessageId)

				m, err := parse(record, traceId)
				if err != nil {
					fmt.Println("[%s] Error:", traceId, err)
					mu.Lock()

					batchItemFailures = append(batchItemFailures, map[string]interface{}{"itemIdentifier": record.MessageId})
					mu.Unlock()

					break
				}

				statusCode, err := callExternalAPI(m.Body, traceId)
				if err != nil {
					mu.Lock()

					batchItemFailures = append(batchItemFailures, map[string]interface{}{"itemIdentifier": m.MessageId})
					mu.Unlock()

					log.Printf("[%s] Error processing message ID %s: %v", traceId, record.MessageId, err)
					break
				}

				if statusCode >= 400 {
					mu.Lock()

					batchItemFailures = append(batchItemFailures, map[string]interface{}{"itemIdentifier": m.MessageId})
					mu.Unlock()

					log.Printf("[%s] API returned error for message ID %s: status code %d", traceId, record.MessageId, statusCode)
					break
				}

				log.Printf("[%s] Successfully processed message ID: %s", traceId, record.MessageId)
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
func callExternalAPI(body Body, traceId string) (int, error) {
	resp, err := http.Post(body.URL, "application/json", bytes.NewBuffer(body.Payload))
	if err != nil {
		return 0, fmt.Errorf("[%s] failed to call external API: %w", traceId, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		responseBody, _ := ioutil.ReadAll(resp.Body)
		log.Printf("[%s] API error: %s", traceId, responseBody)
	}

	return resp.StatusCode, nil
}

func parse(event events.SQSMessage, traceId string) (Message, error) {
	var msg Message
	msg.MessageAttributes = event.MessageAttributes
	msg.MessageId = event.MessageId
	msg.Attributes = event.Attributes

	if val, ok := event.Attributes["MessageGroupID"]; ok {
		msg.MessageGroupID = val
	}

	// Unmarshal the nested JSON in the Body field
	var body Body
	err := json.Unmarshal([]byte(event.Body), &body)
	if err != nil {
		fmt.Println("[%s] Error unmarshalling body JSON:", traceId, err)
		return Message{}, fmt.Errorf("error parsing JSON: %w", err) // Return empty Message and the error
	}
	msg.Body = body

	fmt.Printf("[%s] Message: %+v\n", traceId, msg)
	return msg, nil
}

func main() {
	log.Println("Beginning execution")
	lambda.Start(HandleRequest)
}
