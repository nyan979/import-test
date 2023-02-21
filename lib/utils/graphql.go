package utils

import (
	"net/http"

	"github.com/hasura/go-graphql-client"
	"logur.dev/logur"
)

type GraphqlConf struct {
	HasuraAddress  string
	GraphqlEndpint string
	AdminKey       string
}

func GetGraphqlClient(config GraphqlConf, logger logur.KVLoggerFacade) *graphql.Client {
	logger.Info("--- Connecting to Hasura GraphQL ---")
	graphqlURL := "http://" + config.HasuraAddress + "/" + config.GraphqlEndpint
	client := graphql.NewClient(graphqlURL, nil)
	client = client.WithRequestModifier(func(req *http.Request) {
		req.Header.Set("x-hasura-admin-secret", config.AdminKey)
	})
	logger.Info("Initialized Hasura GraphQL Client:", "hasuraEndpoint",
		config.HasuraAddress, "gqlEndpoint", config.GraphqlEndpint)
	return client
}
