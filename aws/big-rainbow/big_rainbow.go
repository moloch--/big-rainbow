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
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

var (
	bigQueryAPIKey    = os.Getenv("BIGQUERY_API_KEY")
	bigQueryProjectID = os.Getenv("BIGQUERY_PROJECT_ID")
)

// Event -
type Event struct {
	Name string `json:"name"`
}

// RequestHandler - Handle an HTTP request
func RequestHandler(lambdaCtx context.Context, event Event) (string, error) {

	return fmt.Sprintf("Hello %s!", event.Name), nil
}

func main() {
	lambda.Start(RequestHandler)
}
