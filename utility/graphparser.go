package utility

import (
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

type TypeMap map[string]map[string]string

func ParseSchema(schemaStr string) (TypeMap, error) {
	doc, err := gqlparser.LoadSchema(&ast.Source{Input: schemaStr})
	if err != nil {
		return nil, err
	}

	types := make(TypeMap)
	for typeName, def := range doc.Types {
		if validateString(typeName) && len(def.Fields) > 0 {
			fieldMap := make(map[string]string)
			for _, field := range def.Fields {
				if validateString(field.Name) {
					fieldMap[field.Name] = field.Type.String()
				}
			}
			types[typeName] = fieldMap
		}
	}
	return types, nil
}
