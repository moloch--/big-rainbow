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

	"cloud.google.com/go/bigquery"
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

// BigRainbowQuery -
func BigRainbowQuery(hashes QuerySet, projectID string) (ResultSet, error) {

	bigQueryCtx := context.Background()
	bigQueryClient, err := bigquery.NewClient(bigQueryCtx, projectID)

	query := bigQueryClient.Query(`
    SELECT year, SUM(number) as num
    FROM [bigquery-public-data:usa_names.usa_1910_2013]
    WHERE name = "William"
    GROUP BY year
    ORDER BY year`)

	_, err = query.Read(bigQueryCtx)

	if err != nil {
		// TODO: Handle error.
	}

	return ResultSet{}, nil
}
