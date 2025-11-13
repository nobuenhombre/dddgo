package commands

import (
	"go/ast"

	"github.com/nobuenhombre/dddgo/pkg/helpers"
	"github.com/nobuenhombre/suikat/pkg/ge"
)

// Command is a marker type used to identify Command structures.
// Embed this as an anonymous field in your Command structs to enable validation.
type Command struct{}

const (
	DeclaredName = "Command"
	MarkerField  = "_"
	FullPackage  = "github.com/nobuenhombre/dddgo/pkg/layers/infrastructure/interface-adapters/application/objects/commands"
)

// IsCommandTypeDeclaration checks if a struct type contains the Command marker field named "_".
//
// Parameters:
//   - file: The AST file to check imports from
//   - structType: The AST struct type to check
//
// Returns:
//   - true if the struct contains the Command marker named "_", false otherwise
func IsCommandTypeDeclaration(file *ast.File, structType *ast.StructType) bool {
	return helpers.IsSomeObjectTypeDeclaration(file, structType, FullPackage, MarkerField, DeclaredName)
}

// ValidateCommandsReport contains the results of value object validation analysis.
//
// The report provides detailed information about discovered value object types,
// their constructor functions, and any validation violations found during analysis.
//
// Fields:
//   - Types: Map of discovered value object type names to their validation status
//   - Constructors: Map of constructor function names to detailed constructor information
//   - Violations: Map of type names that have validation violations to their violation status
type ValidateCommandsReport struct {
	Types        map[string]bool
	Constructors map[string]*helpers.ConstructorInfo
	Violations   map[string]bool
}

// ValidateCommands analyzes Go source code to validate value object patterns.
//
// This function scans the specified directory for value object type declarations,
// identifies their constructors, and detects potential violations where zero values
// might be improperly initialized.
//
// Parameters:
//   - rootPath: The root directory path to scan for Go source files
//
// Returns:
//   - *ValidateCommandsReport: A detailed report containing found types, constructors, and violations
//   - error: An error if the validation process fails, nil otherwise
//
// The function performs three main steps:
//  1. Discovers value object type declarations in the codebase
//  2. Identifies constructor functions for the discovered types
//  3. Detects violations where zero values might be incorrectly initialized
//
// Returns nil if no value object types are found in the specified directory.
func ValidateCommands(rootPath string) (*ValidateCommandsReport, error) {
	types, err := helpers.FindTypeDeclarations(rootPath, IsCommandTypeDeclaration)
	if err != nil {
		return nil, ge.Pin(err)
	}

	if len(types) == 0 {
		return nil, nil
	}

	constructors, err := helpers.FindConstructors(rootPath, types)
	if err != nil {
		return nil, ge.Pin(err)
	}

	violations, err := helpers.FindZeroValueInitializations(rootPath, DeclaredName, types, constructors)
	if err != nil {
		return nil, ge.Pin(err)
	}

	return &ValidateCommandsReport{
		Types:        types,
		Constructors: constructors,
		Violations:   violations,
	}, nil
}
