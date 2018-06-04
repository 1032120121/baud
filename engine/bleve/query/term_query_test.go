package query

import (
	"testing"
	"encoding/json"
	"github.com/blevesearch/bleve/search/query"
	"reflect"
)

type QueryTestGroup struct {
	input string
	output query.Query
}

func TestParseTermQuery(t *testing.T) {
	groups := []QueryTestGroup{QueryTestGroup{`{
            "status": {
              "value": "urgent",
              "boost": 2.0
            }
          }`,
		func() query.Query {
			utq := query.NewTermQuery("urgent")
			utq.SetField("status")
			utq.SetBoost(2.0)
			return utq
		}(),},
		QueryTestGroup{
			`{
            "status": "normal"
          }`,
			func() query.Query {
				utq := query.NewTermQuery("normal")
				utq.SetField("status")
				utq.SetBoost(1.0)
				return utq
			}(),},
	}

	for _, group := range groups {
		tq := NewTermQuery()
		err := json.Unmarshal([]byte(group.input), tq)
		if err != nil {
			t.Fatal(err)
		}
		ttq, ok := tq.Query.(*query.TermQuery)
		if !ok {
			t.Fatal("parse failed")
		}
		if !reflect.DeepEqual(ttq, group.output) {
			t.Fatalf("parse failed %v %v", ttq, group.output)
		}
	}
}
