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
	"log"
	"strings"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// QuerySet - A set of base64 encoded password hashes to query with
type QuerySet struct {
	Algorithm string   `json:"algorithm"`
	Hashes    []string `json:"hashes"`
}

// Result - Single result of a given hash
type Result struct {
	Preimage string `json:"preimage"`
	Hash     string `json:"hash"`
}

// ResultSet - The set of results from a QuerySet
type ResultSet struct {
	Algorithm string   `json:"algorithm"`
	Results   []Result `json:"results"`
}

// BigQueryMeta - ??
type BigQueryMeta struct {
	ProjectID   string
	Table       string
	Credentials string
}

var (
	supportedAlgorithms = map[string]bool{
		"md5": true,
	}
)

// IsSupportedAlgorithm - Bool
func IsSupportedAlgorithm(algorithm string) bool {
	_, ok := supportedAlgorithms[algorithm]
	return ok
}

// It's up to the caller to make sure these parameters are legit, table is pulled
// from an ENV var and algorithm should be whitelisted, so should be safe from sqli
func getRawQuery(table string, algorithm string, params int) string {
	qParams := "?"
	if 1 < params {
		// BigQuery can't have any trailing ','s
		qParams = strings.Join([]string{qParams, strings.Repeat(", ?", params-1)}, "")
	}
	return fmt.Sprintf("SELECT preimage,%s FROM `%s` WHERE md5 in (%s)", algorithm, table, qParams)
}

// BigRainbowQuery -
func BigRainbowQuery(bigQueryMeta BigQueryMeta, querySet QuerySet) (ResultSet, error) {

	log.Printf("QuerySet = %v", querySet)

	bigQueryCtx := context.Background()
	creds := []byte(bigQueryMeta.Credentials)
	bigQueryClient, err := bigquery.NewClient(bigQueryCtx, bigQueryMeta.ProjectID, option.WithCredentialsJSON(creds))

	rawQuery := getRawQuery(bigQueryMeta.Table, querySet.Algorithm, len(querySet.Hashes))
	log.Printf("RawQuery = %s", rawQuery)
	bigQuery := bigQueryClient.Query(rawQuery)
	var params []bigquery.QueryParameter
	for _, hash := range querySet.Hashes {
		params = append(params, bigquery.QueryParameter{
			Value: hash,
		})
	}
	bigQuery.Parameters = params
	bigQueryResults, err := bigQuery.Read(bigQueryCtx)
	if err != nil {
		return ResultSet{}, err
	}

	resultSet := ResultSet{
		Algorithm: querySet.Algorithm,
		Results:   []Result{},
	}
	for {
		var row []bigquery.Value
		err := bigQueryResults.Next(&row)
		if err == iterator.Done {
			break
		}
		result := Result{
			Preimage: fmt.Sprintf("%s", row[0]),
			Hash:     fmt.Sprintf("%s", row[1]),
		}
		resultSet.Results = append(resultSet.Results, result)
	}

	return resultSet, nil
}
