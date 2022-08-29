package main

import (
	"mssfoobar/ar2-import/lib/test"
)

func main() {
	var gqlClient test.Activities

	gqlClient.GraphQlClient = test.TestImportWorkflow()

	gqlClient.ImportCsvActivity("./lib/data/data.csv")

	// client := graphql.NewClient("http://localhost:8080/v1/graphql", nil)
	// client = client.WithRequestModifier(func(req *http.Request) {
	// 	req.Header.Set("x-hasura-admin-secret", "newadminkey")
	// })

	// csvFile, err := os.Open("./lib/data/data.csv")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// reader := csv.NewReader(csvFile)
	// if _, err := reader.Read(); err != nil {
	// 	log.Fatal(err)
	// }

	// var variables map[string]interface{}

	// type import_csv_insert_input struct {
	// 	Column1  string `json:"column_1,omitempty"`
	// 	Column2  string `json:"column_2,omitempty"`
	// 	Column3  string `json:"column_3,omitempty"`
	// 	Column4  string `json:"column_4,omitempty"`
	// 	Column5  string `json:"column_5,omitempty"`
	// 	Filename string `json:"filename,omitempty"`
	// }

	// var mutation struct {
	// 	InsertData struct {
	// 		Id              graphql.ID
	// 		UploadTimeStamp graphql.String
	// 	} `graphql:"insert_import_csv_one(object: $object)"`
	// }

	// for {
	// 	line, err := reader.Read()
	// 	if err == io.EOF {
	// 		break
	// 	} else if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	variables = map[string]interface{}{
	// 		"object": import_csv_insert_input{
	// 			Column1:  line[0],
	// 			Column2:  line[1],
	// 			Column3:  line[2],
	// 			Column4:  line[3],
	// 			Column5:  line[4],
	// 			Filename: "data.csv",
	// 		},
	// 	}

	// 	if err := client.Mutate(context.Background(), &mutation, variables); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	// csvJson, _ := json.Marshal(csvData)
	// fmt.Println(string(csvJson))

	// type import_csv_insert_input struct {
	// 	Id      string `json:"id,omitempty"`
	// 	Report1 string `json:"report_1,omitempty"`
	// 	Report2 string `json:"report_2,omitempty"`
	// }

	// var mutation struct {
	// 	InsertData struct {
	// 		Id graphql.ID
	// 	} `graphql:"insert_import_csv_one(object: $object)"`
	// }

	// variables := map[string]interface{}{
	// 	"object": import_csv_insert_input{
	// 		Report1: "asdfsfa",
	// 		Report2: "sdfsdfsfs",
	// 	},
	// }

	// if err := client.Mutate(context.Background(), &mutation, variables); err != nil {
	// 	panic(err)
	// }

	// var q struct {
	// 	Import_csv []struct {
	// 		Report_1 graphql.String
	// 	}
	// }

	// variables := map[string]interface{}{
	// 	"characterID": graphql.ID("1"),
	// }

	// err := client.Query(context.Background(), &q, nil)
	// if err != nil {
	// 	panic(err)
	// }

	// data := q.Import_csv

	// fmt.Println(data)
}
