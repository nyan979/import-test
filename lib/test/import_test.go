package test

import (
	"context"
	"net/http"

	"github.com/hasura/go-graphql-client"
)

type Activities struct {
	GraphQlClient *graphql.Client
}

type csv struct {
	Id              string `json:"id,omitempty"`
	Column1         string `json:"column_1,omitempty"`
	Column2         string `json:"column_2,omitempty"`
	Column3         string `json:"column_3,omitempty"`
	Column4         string `json:"column_4,omitempty"`
	Column5         string `json:"column_5,omitempty"`
	UploadTimeStamp string `json:"uploadTimeStamp,omitempty"`
	Filename        string `json:"filename,omitempty"`
}

func TestImportWorkflow() *graphql.Client {
	client := graphql.NewClient("http://localhost:8080/v1/graphql", nil)
	client = client.WithRequestModifier(func(req *http.Request) {
		req.Header.Set("x-hasura-admin-secret", "newadminkey")
	})

	return client
}

var activities *Activities

type input csv

func (a *Activities) ImportCsvActivity(variables map[string]interface{}) error {
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
			Id graphql.ID
		} `graphql:"insert_import_csv(object: $object)"`
	}

	variables = map[string]interface{}{
		"object": import_csv_insert_input{},
	}

	if err := a.GraphQlClient.Mutate(context.Background(), &mutation, variables); err != nil {
		return err
	}

	return nil
}
