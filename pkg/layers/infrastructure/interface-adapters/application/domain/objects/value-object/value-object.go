// Package valueobject provides validation for DDD and Clean Architecture semantics
// to ensure domain purity of Value Objects.
//
// Value Objects are identified by embedding the ValueObject marker type:
//
//		type Location struct {
//	        x int
//	        y int
//	        ...
//			_ valueobject.ValueObject
//		}
//
// The package scans Go source files and detects violations where Value Objects
// are initialized with zero values outside their constructor functions.
//
// Example Test Code
//
//	func TestIsValueObjectsValidSemantics(t *testing.T) {
//		 projectRoot, err := helpers.FindProjectRoot()
//		 assert.NoError(t, err)
//
//		 report, err := ValidateValueObjects(projectRoot)
//		 assert.NoError(t, err)
//
//		 if report == nil {
//			 t.Skip("no value objects found")
//		 }
//
//		 for typeDeclaration := range report.Types {
//			 t.Logf("found declared Value Object: %s", typeDeclaration)
//		 }
//
//		 for typeDeclaration, constructor := range report.Constructors {
//			 t.Logf(
//				 "found Value Object [%s] constructor: %s:%d:%d",
//				 typeDeclaration,
//				 constructor.File,
//				 constructor.StartLine,
//				 constructor.EndLine,
//			 )
//		 }
//
//		 for violation, _ := range report.Violations {
//			 t.Logf("VIOLATION: %s", violation)
//		 }
//
//		 assert.Equal(t, 0, len(report.Violations))
//	}
package valueobject

import (
	"go/ast"

	"github.com/nobuenhombre/dddgo/pkg/helpers"
	"github.com/nobuenhombre/suikat/pkg/ge"
)

// ValueObject is a marker type used to identify Value Object structures.
// Embed this as an anonymous field in your Value Object structs to enable validation.
type ValueObject struct{}

const (
	DeclaredName = "ValueObject"
	MarkerField  = "_"
	FullPackage  = "github.com/nobuenhombre/dddgo/pkg/layers/infrastructure/interface-adapters/application/domain/objects/value-object"
)

// IsValueObjectTypeDeclaration checks if a struct type contains the ValueObject marker field named "_".
//
// Parameters:
//   - file: The AST file to check imports from
//   - structType: The AST struct type to check
//
// Returns:
//   - true if the struct contains the ValueObject marker named "_", false otherwise
func IsValueObjectTypeDeclaration(file *ast.File, structType *ast.StructType) bool {
	return helpers.IsSomeObjectTypeDeclaration(file, structType, FullPackage, MarkerField, DeclaredName)
}

// ValidateValueObjectsReport contains the results of value object validation analysis.
//
// The report provides detailed information about discovered value object types,
// their constructor functions, and any validation violations found during analysis.
//
// Fields:
//   - Types: Map of discovered value object type names to their validation status
//   - Constructors: Map of constructor function names to detailed constructor information
//   - Violations: Map of type names that have validation violations to their violation status
type ValidateValueObjectsReport struct {
	Types        map[string]bool
	Constructors map[string]*helpers.ConstructorInfo
	Violations   map[string]bool
}

// ValidateValueObjects analyzes Go source code to validate value object patterns.
//
// This function scans the specified directory for value object type declarations,
// identifies their constructors, and detects potential violations where zero values
// might be improperly initialized.
//
// Parameters:
//   - rootPath: The root directory path to scan for Go source files
//
// Returns:
//   - *ValidateValueObjectsReport: A detailed report containing found types, constructors, and violations
//   - error: An error if the validation process fails, nil otherwise
//
// The function performs three main steps:
//  1. Discovers value object type declarations in the codebase
//  2. Identifies constructor functions for the discovered types
//  3. Detects violations where zero values might be incorrectly initialized
//
// Returns nil if no value object types are found in the specified directory.
func ValidateValueObjects(rootPath string) (*ValidateValueObjectsReport, error) {
	types, err := helpers.FindTypeDeclarations(rootPath, IsValueObjectTypeDeclaration)
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

	return &ValidateValueObjectsReport{
		Types:        types,
		Constructors: constructors,
		Violations:   violations,
	}, nil
}
