package sqlast

type Node interface {
	SQLString(cfg *SQLGenerator) string
}

// BooleanNode is a node that returns a boolean value when evaluated.
type BooleanNode interface {
	Node
}

type RegoSource string

type invalidNode struct{}

func (i invalidNode) SQLString(cfg *SQLGenerator) string {
	return "invalid_type"
}

// IsPrimitive is a nice helper function to cover the most common primitive
// types.
func IsPrimitive(v Node) bool {
	switch v.(type) {
	case number, astString, boolean, BooleanNode:
		return true
	}
	return false
}

//func IsLiteral(v Node) bool {
//	switch v.(type) {
//	case number, astString, boolean:
//		return true
//	}
//	return false
//}
