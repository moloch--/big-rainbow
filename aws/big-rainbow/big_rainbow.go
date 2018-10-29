package main

/*
	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var (
	bigQueryMeta = BigQueryMeta{
		Table:       os.Getenv("BIGQUERY_TABLE"),
		ProjectID:   os.Getenv("BIGQUERY_PROJECT_ID"),
		Credentials: os.Getenv("BIGQUERY_CREDENTIALS"),
	}

	// Generic Errors
	errNoBody               = errors.New("no HTTP body")
	errFailedToParseBody    = errors.New("failed to parse HTTP body")
	errNoHashes             = errors.New("No hashes in request")
	errUnsupportedAlgorithm = errors.New("Unsupported hash algorithm")
)

// LambdaError - Error mapped to JSON
type LambdaError struct {
	Error string `json:"error"`
}

// JSONError - Returns an error formatted as an APIGatewayProxyResponse
// if you try to return an actual error the API Gateway just swaps it
// for a generic 500 because why the fuck would you just expect an error
// to get returned to the client if you explicitly return it
func JSONError(err error) events.APIGatewayProxyResponse {
	msg, _ := json.Marshal(LambdaError{
		Error: fmt.Sprintf("%v", err),
	})
	return events.APIGatewayProxyResponse{
		StatusCode: 400,
		Body:       string(msg),
	}
}

// RequestHandler - Handle an HTTP request
func RequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	log.Printf("Processing Lambda request %s\n", request.RequestContext.RequestID)

	// If no name is provided in the HTTP request body, throw an error
	if len(request.Body) < 1 {
		return JSONError(errNoBody), nil
	}

	log.Printf("Parsing request body ...")
	var querySet QuerySet
	err := json.Unmarshal([]byte(request.Body), &querySet)
	if err != nil {
		return JSONError(errFailedToParseBody), nil
	}

	if !IsSupportedAlgorithm(querySet.Algorithm) {
		return JSONError(errUnsupportedAlgorithm), nil
	}

	unique(&querySet)
	if len(querySet.Hashes) == 0 {
		return JSONError(errNoHashes), nil
	}

	resultSet, err := BigRainbowQuery(bigQueryMeta, querySet)
	if err != nil {
		return JSONError(err), nil
	}

	response, err := json.Marshal(resultSet)
	if err != nil {
		return JSONError(err), nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(response),
		StatusCode: 200,
	}, nil
}

// Remove any duplicate/blank hashes
func unique(querySet *QuerySet) {
	uniqueValues := make(map[string]bool)
	for _, value := range querySet.Hashes {
		if 0 < len(value) {
			uniqueValues[value] = true
		}
	}
	var keys []string
	for key := range uniqueValues {
		keys = append(keys, key)
	}
	querySet.Hashes = keys
}

func main() {
	lambda.Start(RequestHandler)
}
