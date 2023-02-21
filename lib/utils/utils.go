package utils

import (
	"context"
	"fmt"
	"log"
	"mssfoobar/ar2-import/workflows/import/workflow"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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

func InitZapLogger(loggingLevel string) *zap.Logger {
	// Default to production level
	logLevel := zap.InfoLevel
	isDev := false
	// Set development to TRUE if DEVELOPMENT is set to true, otherwise default to false
	if loggingLevel == "debug" {
		isDev = true
		logLevel = zap.DebugLevel
	}
	encodeConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(logLevel),
		Development:      isDev,
		Sampling:         nil, // consider exposing this to config
		Encoding:         "console",
		EncoderConfig:    encodeConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	logger, err := config.Build()
	if err != nil {
		log.Fatal("Unable to create zap logger")
	}
	return logger
}

// initialize temporal import service workflow
func CreateImportWorkflow(c client.Client, signal *workflow.ImportSignal) error {
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "import-service",
		ID:        signal.Message.RequestID,
	}
	ctx := context.Background()
	_, err := c.ExecuteWorkflow(ctx, workflowOptions, workflow.ImportServiceWorkflow)
	if err != nil {
		return fmt.Errorf("unable to execute order workflow: %w", err)
	}
	return nil
}

// execute external workflow to send/recieve signal channel from/to import service workflow
func ExecuteImportWorkflow(c client.Client, requestId string, signal *workflow.ImportSignal) (*workflow.ImportSignal, error) {
	workflowOptions := client.StartWorkflowOptions{
		TaskQueue: "import-service",
	}
	ctx := context.Background()
	we, err := c.ExecuteWorkflow(ctx, workflowOptions, workflow.SignalImportServiceWorkflow, requestId, signal)
	if err != nil {
		return signal, fmt.Errorf("unable to execute workflow: %w", err)
	}
	err = we.Get(ctx, &signal)
	if err != nil {
		return signal, err
	}
	return signal, nil
}
