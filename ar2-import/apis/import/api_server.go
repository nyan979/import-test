package main

import (
	"log"
	"mssfoobar/ar2-import/ar2-import/lib/utils"
	"mssfoobar/ar2-import/ar2-import/lib/workflow"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

type Application struct {
	activities workflow.Activities
}

func main() {
	godotenv.Load("../../../.env")

	// TODO: implement temporal workflow
	app := Application{
		activities: workflow.Activities{
			MinioClient: utils.SetMinioClient(),
			// GraphqlClient: utils.SetGraphqlClient(),
			// KafkaReader:   utils.NewKafkaReader(),
			// KafkaWriter:   utils.NewKafkaWriter(),
			RequestId: "",
		},
	}

	// ctx := context.Background()
	// // RequestId = make(chan string, 1000)
	// message := make(chan kafka.Message, 1000)
	// messageCommit := make(chan kafka.Message, 1000)

	// g, ctx := errgroup.WithContext(ctx)

	// // fetch minio notification message go routine
	// g.Go(func() error {
	// 	return app.activities.FetchMessage(ctx, message)
	// })

	// // write csv content to kafka topic go routine
	// g.Go(func() error {
	// 	return app.activities.WriteMessages(ctx, message, messageCommit /*, RequestId*/)
	// })

	// // commit to offset minio notification messages go routine
	// g.Go(func() error {
	// 	return app.activities.CommitMessages(ctx, messageCommit)
	// })

	// set and serve on port
	port := ":" + os.Getenv("APP_PORT")

	err := http.ListenAndServe(port, app.routes())
	if err != nil {
		log.Fatalln(err)
	}

	// conErr := g.Wait()
	// if conErr != nil {
	// 	log.Fatalln(conErr)
	// }
}
