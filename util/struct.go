package util

import (
	"encoding/json"
)

func StructToMap(o any) (map[string]any, error) {
	doc, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	m := &map[string]interface{}{}
	err = json.Unmarshal(doc, m)
	if err != nil {
		return nil, err
	}
	return *m, nil
}

func MustStructToMap(o any) map[string]any {
	m, err := StructToMap(o)
	if err != nil {
		panic(err)
	}
	return m
}

func MapToStruct(m map[string]any, o any) error {
	doc, err := json.Marshal(m)
	if err != nil {
		return err
	}

	err = json.Unmarshal(doc, o)
	if err != nil {
		return err
	}
	return nil
}

func MustMapToStruct(m map[string]any, o any) {
	if err := MapToStruct(m, o); err != nil {
		panic(err)
	}
}
