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

// BigQueryMeta - BigQuery metadata info
type BigQueryMeta struct {
	ProjectID   string
	Table       string
	Credentials string
}

var (
	supportedAlgorithms = map[string]bool{
		"md4":      true,
		"md5":      true,
		"sha1":     true,
		"sha2_224": true,
		"sha2_256": true,
		"sha2_384": true,
		"sha2_512": true,
		"sha3_224": true,
		"sha3_256": true,
		"sha3_384": true,
		"sha3_512": true,
		// "ripemd160":             true,
		"lm":   true,
		"ntlm": true,
		// "mysql323":              true,
		// "mysql41":               true,
		// "oracle10g_sys":         true,
		// "oracle10g_system":      true,
		// "msdcc_administrator":   true,
		// "msdcc2_administrator":  true,
		// "postgres_md5_admin":    true,
		// "postgres_md5_postgres": true,
		// "postgres_md5_root":     true,
		// "whirlpool":             true,
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
	return fmt.Sprintf("SELECT preimage,%s FROM `%s` WHERE %s in (%s)",
		algorithm, table, algorithm, qParams)
}

// BigRainbowQuery - Execute a BigQuery query searaching for a querySet
func BigRainbowQuery(bigQueryMeta BigQueryMeta, querySet QuerySet) (ResultSet, error) {

	bigQueryCtx := context.Background()
	options := option.WithCredentialsJSON([]byte(bigQueryMeta.Credentials))
	bigQueryClient, err := bigquery.NewClient(bigQueryCtx, bigQueryMeta.ProjectID, options)

	rawQuery := getRawQuery(bigQueryMeta.Table, querySet.Algorithm, len(querySet.Hashes))

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
