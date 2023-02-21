package utils

// func InitMinioKafkaReader(logger temporalLog.Logger) *kafka.Reader {
// 	host := os.Getenv("KAFKA_HOST")
// 	port := os.Getenv("KAFKA_PORT")
// 	topic := os.Getenv("KAFKA_MINIO_TOPIC")

// 	reader := kafka.NewReader(kafka.ReaderConfig{
// 		Brokers: []string{host + ":" + port},
// 		Topic:   topic,
// 		GroupID: "testing",
// 	})

// 	logger.Info("Initialized Minio Kafka Reader:", "brokerHost", host, "brokerPort", port, "topic", topic)

// 	return reader
// }

// func InitImportKafkaReader(logger temporalLog.Logger) *kafka.Reader {
// 	host := os.Getenv("KAFKA_HOST")
// 	port := os.Getenv("KAFKA_PORT")
// 	topic := os.Getenv("KAFKA_SERVICE_OUT_TOPIC")

// 	reader := kafka.NewReader(kafka.ReaderConfig{
// 		Brokers: []string{host + ":" + port},
// 		Topic:   topic,
// 		GroupID: "testing",
// 	})

// 	logger.Info("Initialized Service_Out Kafka Reader:", "brokerHost", host, "brokerPort", port, "topic", topic)

// 	return reader
// }

// func InitKafkaWriter(logger temporalLog.Logger) *kafka.Writer {
// 	host := os.Getenv("KAFKA_HOST")
// 	port := os.Getenv("KAFKA_PORT")
// 	topic := os.Getenv("KAFKA_SERVICE_IN_TOPIC")

// 	writer := &kafka.Writer{
// 		Addr:  kafka.TCP(host + ":" + port),
// 		Topic: topic,
// 	}

// 	logger.Info("Initialized Service_In Kafka Writer:", "brokerHost", host, "brokerPort", port, "topic", topic)

// 	return writer
// }
