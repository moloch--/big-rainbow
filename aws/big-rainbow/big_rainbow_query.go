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
	"log"

	"cloud.google.com/go/bigquery"
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
	DatasetID   string
	TableID     string
	Credentials string
}

// BigRainbowQuery -
func BigRainbowQuery(bigQueryMeta BigQueryMeta, hashes QuerySet) (ResultSet, error) {

	log.Printf("QuerySet = %v", hashes)

	bigQueryCtx := context.Background()
	creds := []byte(bigQueryMeta.Credentials)
	bigQueryClient, err := bigquery.NewClient(bigQueryCtx, bigQueryMeta.ProjectID, option.WithCredentialsJSON(creds))

	bigQuery := bigQueryClient.Query(`
		SELECT primage,md5
		FROM [rainbow1.crackstation_human_only]
		WHERE md5 = 'X03MO1qnZdYdgyfeuILPmQ=='`)

	results, err := bigQuery.Read(bigQueryCtx)

	log.Printf("Results = %v", results)

	if err != nil {
		return ResultSet{}, err
	}

	return ResultSet{}, nil
}
