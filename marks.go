package rego2sql

import "github.com/zclconf/go-cty/cty"

const (
	markJSONB      = "jsonb"
	markUnknownRef = "unknown-ref"
)

func MarkJSONB(v cty.Value) cty.Value {
	return v.Mark(markJSONB)
}

func IsJSONBool(v cty.Value) bool {
	return v.HasMark(markJSONB)
}
