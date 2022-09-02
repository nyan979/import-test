package main

import "mssfoobar/ar2-import/lib/test"

func main() {
	var gqlClient test.Activities
	var minioClient MinioClient

	gqlClient.GraphQlClient = test.TestImportWorkflow()
	minioClient.client = setMinioClient()

	//gqlClient.ImportCsvActivity("./lib/data/data.csv")
	minioClient.SetupRoutes()

}
