package test

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/hasura/go-graphql-client"
)

type Activities struct {
	GraphQlClient *graphql.Client
}

func TestImportWorkflow() *graphql.Client {
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	gqlEndpoint := os.Getenv("GQL_ENDPOINT")
	adminkey := os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET")

	graphqlURL := "http://" + dbHost + ":" + dbPort + "/" + gqlEndpoint

	client := graphql.NewClient(graphqlURL, nil)
	client = client.WithRequestModifier(func(req *http.Request) {
		req.Header.Set("x-hasura-admin-secret", adminkey)
	})

	return client
}

func (a *Activities) ImportCsvActivity(filepath string) error {
	type import_csv_insert_input struct {
		Id              string `json:"id,omitempty"`
		Column1         string `json:"column_1,omitempty"`
		Column2         string `json:"column_2,omitempty"`
		Column3         string `json:"column_3,omitempty"`
		Column4         string `json:"column_4,omitempty"`
		Column5         string `json:"column_5,omitempty"`
		UploadTimeStamp string `json:"uploadTimeStamp,omitempty"`
		Filename        string `json:"filename,omitempty"`
	}

	var mutation struct {
		InsertData struct {
			Id              graphql.ID
			UploadTimeStamp graphql.String
		} `graphql:"insert_import_csv_one(object: $object)"`
	}

	reader := LoadCSV(filepath)
	var variables map[string]interface{}

	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		variables = map[string]interface{}{
			"object": import_csv_insert_input{
				Column1:  line[0],
				Column2:  line[1],
				Column3:  line[2],
				Column4:  line[3],
				Column5:  line[4],
				Filename: "data.csv",
			},
		}

		if err := a.GraphQlClient.Mutate(context.Background(), &mutation, variables); err != nil {
			log.Fatal(err)
		}
	}

	return nil
}
