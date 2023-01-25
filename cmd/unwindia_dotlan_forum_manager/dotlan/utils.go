package dotlan

import (
	"errors"
	"fmt"
	"github.com/GSH-LAN/Unwindia_common/src/go/helper"
	"github.com/GSH-LAN/Unwindia_common/src/go/sql"
	"github.com/rs/zerolog/log"
	"reflect"
)

func (d *DotlanDbClientImpl) getFieldsFromModelWithoutTablename(model sql.Table) (tableName string, fieldList string, err error) {
	if tabler, ok := model.(sql.Table); ok {
		tableName = tabler.TableName()
	} else {
		return "", "", errors.New("model is not a table")
	}
	fieldList, err = d.getFieldsFromModelWithTablename(model, tableName)
	return
}

func (d *DotlanDbClientImpl) getFieldsFromModelWithTablename(model interface{}, tableName string) (string, error) {

	if queryString, ok := d.modelFieldCache[tableName]; ok {
		log.Debug().Str("table", tableName).Str("query", queryString).Msg("Retrieved query from cache")
		return queryString, nil
	}

	var columns []string
	var queryString string

	qry := fmt.Sprintf("SELECT column_name FROM information_schema.columns WHERE table_name = '%s'", tableName)
	log.Trace().Str("query", qry).Msgf("getFieldsFromModelWithTablename")
	if err := d.db.Select(&columns, qry); err != nil {
		return "", err
	}

	reflectedModel := reflect.ValueOf(model)

	for i := 0; i < reflectedModel.NumField(); i++ {
		typeField := reflectedModel.Type().Field(i)
		if val, ok := typeField.Tag.Lookup("db"); ok {
			if helper.StringSliceContains(columns, val) {
				if len(queryString) > 0 {
					queryString = queryString + ", " + val
				} else {
					queryString = val
				}
			}
		}
	}

	d.modelFieldCache[tableName] = queryString

	return queryString, nil
}
