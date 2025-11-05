package entity

import (
	"go/ast"

	"github.com/nobuenhombre/dddgo/pkg/helpers"
)

type Entity struct{}

const (
	DeclaredName = "Entity"
	MarkerField  = "_"
	FullPackage  = "github.com/nobuenhombre/dddgo/pkg/layers/infrastructure/interface-adapters/application/domain/objects/entity"
)

// IsEntityTypeDeclaration checks if a struct type contains the Entity marker field named "_".
//
// Parameters:
//   - file: The AST file to check imports from
//   - structType: The AST struct type to check
//
// Returns:
//   - true if the struct contains the Entity marker named "_", false otherwise
func IsEntityTypeDeclaration(file *ast.File, structType *ast.StructType) bool {
	return helpers.IsSomeObjectTypeDeclaration(file, structType, FullPackage, MarkerField, DeclaredName)
}
