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
		Credentials: os.Getenv("BIGQUERY_CREDENTIALS"),
	}

	// Generic Errors
	errNoBody            = errors.New("no HTTP body")
	errFailedToParseBody = errors.New("failed to parse HTTP body")
	errNoHashes          = errors.New("No hashes in request")
)

// LambdaError - Error mapped to JSON
type LambdaError struct {
	Error string `json:"error"`
}

// JSONError - Returns an error formatted as an APIGatewayProxyResponse
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

func main() {
	lambda.Start(RequestHandler)
}
