package workflow

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

// func (a *Activities) ReadMessage(ctx context.Context, messages chan<- kafka.Message, messageWrite chan string) error {
// 	log.Println("Kafka Read Message Starting...")
// 	for {
// 		message, err := a.KafkaReader.ReadMessage(ctx)
// 		if err != nil {
// 			return err
// 		}

// 		select {
// 		case <-ctx.Done():
// 			return ctx.Err()
// 		case messages <- message:
// 			filename, err := a.ReadMinioNotification(message)
// 			if err != nil {
// 				return err
// 			}
// 			lines, err := a.ParseCSVToLine(filename)
// 			if err != nil {
// 				return err
// 			}
// 			for _, v := range lines {
// 				messageWrite <- v
// 				log.Printf("message sent to a channel: %v \n", string(v))
// 			}
// 		}
// 	}
// }

// func (a *Activities) WriteMessages(ctx context.Context, messageWrite chan string, messageCommit chan string) error {
// 	log.Println("Kafka Write Message Starting...")
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return ctx.Err()
// 		case str := <-messageWrite:
// 			err := a.KafkaWriter.WriteMessages(ctx, kafka.Message{
// 				Value: []byte(str),
// 			})
// 			if err != nil {
// 				return err
// 			}
// 		}
// 	}
// }

func (a *Activities) FetchMessage(ctx context.Context, messages chan<- kafka.Message) error {
	log.Println("Kafka Fetch Message Starting...")
	for {
		message, err := a.KafkaReader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case messages <- message:
		}
	}
}

func (a *Activities) WriteMessages(ctx context.Context, messages chan kafka.Message, messagesCommit chan kafka.Message) error {
	log.Println("Kafka Write Message Starting...")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-messages:
			filename, err := a.ReadMinioNotification(msg)
			if err != nil {
				return err
			}
			lines, err := a.ParseCSVToLine(filename)
			if err != nil {
				return err
			}
			var kafkaMessage []kafka.Message
			for _, v := range lines {
				kafkaMessage = append(kafkaMessage, kafka.Message{
					Value: []byte(v),
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
