package aggregate

import (
	"go/ast"

	"github.com/nobuenhombre/dddgo/pkg/helpers"
)

type Aggregate struct{}

type AggregateRoot struct{}

const (
	DeclaredName     = "Aggregate"
	DeclaredRootName = "AggregateRoot"
	MarkerField      = "_"
	FullPackage      = "github.com/nobuenhombre/dddgo/pkg/layers/infrastructure/interface-adapters/application/domain/objects/aggregate"
)

// IsAggregateTypeDeclaration checks if a struct type contains the Aggregate marker field named "_".
//
// Parameters:
//   - file: The AST file to check imports from
//   - structType: The AST struct type to check
//
// Returns:
//   - true if the struct contains the Aggregate marker named "_", false otherwise
func IsAggregateTypeDeclaration(file *ast.File, structType *ast.StructType) bool {
	return helpers.IsSomeObjectTypeDeclaration(file, structType, FullPackage, MarkerField, DeclaredName)
}

// IsAggregateRootTypeDeclaration checks if a struct type contains the AggregateRoot marker field named "_".
//
// Parameters:
//   - file: The AST file to check imports from
//   - structType: The AST struct type to check
//
// Returns:
//   - true if the struct contains the AggregateRoot marker named "_", false otherwise
func IsAggregateRootTypeDeclaration(file *ast.File, structType *ast.StructType) bool {
	return helpers.IsSomeObjectTypeDeclaration(file, structType, FullPackage, MarkerField, DeclaredRootName)
}
