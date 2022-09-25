package test

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type jsonMessage struct {
	RequestId string `json:"requestId"`
	Record    string `json:"record"`
}

func (a *Activities) FetchMessage(ctx context.Context, messages chan<- kafka.Message) error {
	log.Println("Kafka Fetch Message Starting...")
	for {
		msg, err := a.KafkaReader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case messages <- msg:
			log.Printf("Fetched an msg: %v \n", string(msg.Value))
		}
	}
}

func (a *Activities) WriteMessages(ctx context.Context, messages <-chan kafka.Message, messagesCommit chan<- kafka.Message) error {
	log.Println("Kafka Write Message Starting...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-messages:
			requestId := "a.RequestId"
			minioMsg, err := a.ReadMinioNotification(msg)
			if err != nil {
				a.UpdateConfigRunTimeStatus(requestId, "failed")
				return err
			}

			filename := minioMsg.Records[0].S3.Object.Key
			fileversion := minioMsg.Records[0].S3.Object.VersionId

			err = a.UpdateConfigRunTimeFileVersion(requestId, fileversion)
			if err != nil {
				a.UpdateConfigRunTimeStatus(requestId, "failed")
				return err
			}

			err = a.UpdateConfigRunTimeStatus(requestId, "Importing")
			if err != nil {
				a.UpdateConfigRunTimeStatus(requestId, "failed")
				return err
			}

			lines, err := a.ParseCSVToLine(filename)
			if err != nil {
				a.UpdateConfigRunTimeStatus(requestId, "failed")
				return err
			}

			var jsonMessages []jsonMessage

			for _, line := range lines {
				jsonMessages = append(jsonMessages, jsonMessage{
					RequestId: requestId,
					Record:    line,
				})
			}

			var kafkaMessage []kafka.Message
			for _, msg := range jsonMessages {
				jsonByte, _ := json.Marshal(msg)
				kafkaMessage = append(kafkaMessage, kafka.Message{
					Value: jsonByte,
				})
			}

			err = a.KafkaWriter.WriteMessages(ctx, kafkaMessage...)
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
			case messagesCommit <- msg:
			}
		}
	}
}

func (a *Activities) CommitMessages(ctx context.Context, messagesCommit <-chan kafka.Message) error {
	for {
		select {
		case <-ctx.Done():
		case msg := <-messagesCommit:
			err := a.KafkaReader.CommitMessages(ctx, msg)
			if err != nil {
				return err
			}
			log.Printf("committed an msg: %v \n", string(msg.Value))
		}
	}
}
