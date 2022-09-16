package workflow

import (
	"context"
	"log"
	"os"

	"github.com/segmentio/kafka-go"
)

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

func (a *Activities) WriteMessages(ctx context.Context, messages <-chan kafka.Message, messagesCommit chan<- kafka.Message, RequestId <-chan string) error {
	log.Println("Kafka Write Message Starting...")

	address := os.Getenv("KAFKA_HOST") + ":" + os.Getenv("KAFKA_PORT")
	topic := os.Getenv("KAFKA_SERVICE_TOPIC")
	counter := 0

	conn, err := kafka.Dial("tcp", address)
	if err != nil {
		return err
	}

	partition, _ := conn.ReadPartitions(topic)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-messages:
			requestId := <-RequestId
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

			var kafkaMessage []kafka.Message
			for _, line := range lines {
				kafkaMessage = append(kafkaMessage, kafka.Message{
					Key:   []byte(requestId),
					Value: []byte(line),
				})
			}

			conn, err := kafka.DialLeader(ctx, "tcp", address, topic, partition[counter].ID)
			if err != nil {
				a.UpdateConfigRunTimeStatus(requestId, "failed")
				return err
			}

			_, err = conn.WriteMessages(kafkaMessage...)
			if err != nil {
				a.UpdateConfigRunTimeStatus(requestId, "failed")
				return err
			}

			counter = (counter + 1) % len(partition)

			// err = a.KafkaWriter.WriteMessages(ctx, kafkaMessage...)
			// if err != nil {
			// 	return err
			// }

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
