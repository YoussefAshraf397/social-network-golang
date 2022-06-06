package service

import (
	"bytes"
	"fmt"
	"github.com/jackc/pgx"
	"html/template"
	"strings"
)

const (
	minPageSize     = 1
	defaultPageSize = 10
	maxPageSize     = 99
)

var queriesCache = make(map[string]*template.Template)

func isUniqueViolation(err error) bool {
	pgerror, ok := err.(pgx.PgError)
	return ok && pgerror.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	pgerror, ok := err.(pgx.PgError)
	return ok && pgerror.Code == "23503"
}

func buildQuery(text string, data map[string]interface{}) (string, []interface{}, error) {

	t, ok := queriesCache[text]
	if !ok {
		var err error
		t, err = template.New("query").Parse(text)
		if err != nil {
			return "", nil, fmt.Errorf("could not parse sql query template %v", err)
		}
		queriesCache[text] = t
	}

	var wr bytes.Buffer
	if err := t.Execute(&wr, data); err != nil {
		return "", nil, fmt.Errorf("could not apply sql query data %v", err)
	}

	query := wr.String()
	args := []interface{}{}
	for key, val := range data {
		if !strings.Contains(query, "@"+key) {
			continue
		}
		args = append(args, val)
		query = strings.ReplaceAll(query, "@"+key, fmt.Sprintf("$%d", len(args)))
	}
	return query, args, nil
}

func normalizePageSize(i int) int {
	if i == 0 {
		return defaultPageSize
	}
	if i < minPageSize {
		return minPageSize
	}
	if i > maxPageSize {
		return maxPageSize

	}
	return i
}
