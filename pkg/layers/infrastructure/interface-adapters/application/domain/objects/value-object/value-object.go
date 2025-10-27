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
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/nobuenhombre/suikat/pkg/ge"
)

// ValueObject is a marker type used to identify Value Object structures.
// Embed this as an anonymous field in your Value Object structs to enable validation.
type ValueObject struct{}

const (
	DeclaredName = "ValueObject"
	MarkerField  = "_"
	FullPackage  = "github.com/nobuenhombre/dddgo/pkg/layers/infrastructure/interface-adapters/application/domain/objects/value-object/valueobject"
)

// isValueObjectTypeDeclaration checks if a struct type contains the ValueObject marker field named "_".
//
// Parameters:
//   - file: The AST file to check imports from
//   - structType: The AST struct type to check
//
// Returns:
//   - true if the struct contains the ValueObject marker named "_", false otherwise
func isValueObjectTypeDeclaration(file *ast.File, structType *ast.StructType) bool {
	if structType.Fields == nil {
		return false
	}

	pkgAlias := getPackageAlias(file, FullPackage)
	if pkgAlias == "" {
		return false
	}

	for _, field := range structType.Fields.List {
		// STRICT CHECK: Only fields explicitly named "_" are considered Value Object markers
		if len(field.Names) == 1 && field.Names[0].Name == MarkerField {
			if selector, ok := field.Type.(*ast.SelectorExpr); ok {
				if ident, ok := selector.X.(*ast.Ident); ok {
					if ident.Name == pkgAlias && selector.Sel.Name == DeclaredName {
						return true
					}
				}
			}
		}
	}

	return false
}

// getPackageAlias finds the package alias for a given full package path in the file's imports.
//
// Parameters:
//   - file: The AST file to check imports from
//   - fullPackagePath: The full package path to look for
//
// Returns:
//   - The package alias if found, empty string otherwise
func getPackageAlias(file *ast.File, fullPackagePath string) string {
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		if importPath == fullPackagePath {
			if imp.Name != nil {
				return imp.Name.Name
			}

			parts := strings.Split(fullPackagePath, "/")
			return parts[len(parts)-1] // "valueobject"
		}
	}
	return ""
}

// findValueObjectTypeDeclarations scans the project directory for Value Object type declarations.
//
// Parameters:
//   - rootPath: The root directory path to scan for Go files
//
// Returns:
//   - A map of Value Object type names to boolean values indicating their presence
//   - An error if the scan fails, nil otherwise
func findValueObjectTypeDeclarations(rootPath string) (map[string]bool, error) {
	valueObjectsTypes := make(map[string]bool)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || filepath.Ext(path) != ".go" {
			return nil
		}

		// Skip test files - we intentionally allow zero-value initializations in tests
		// to provide flexibility for testing scenarios that don't require full domain validation
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}

		ast.Inspect(file, func(n ast.Node) bool {
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return true
			}

			if isValueObjectTypeDeclaration(file, structType) {
				valueObjectsTypes[typeSpec.Name.Name] = true
			}

			return true
		})

		return nil
	})

	if err != nil {
		return nil, ge.Pin(err)
	}

	return valueObjectsTypes, nil
}

// ConstructorInfo contains location information about a Value Object constructor function.
type ConstructorInfo struct {
	File      string
	StartLine int
	EndLine   int
}

// findConstructors locates all constructor functions for Value Objects in the project.
//
// Parameters:
//   - rootPath: The root directory path to scan for Go files
//   - voTypes: A map of Value Object type names to search constructors for
//
// Returns:
//   - A map of constructor names to their location information
//   - An error if the scan fails, nil otherwise
func findConstructors(rootPath string, voTypes map[string]bool) (map[string]*ConstructorInfo, error) {
	constructors := make(map[string]*ConstructorInfo)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || filepath.Ext(path) != ".go" {
			return nil
		}

		// Skip test files - we intentionally allow zero-value initializations in tests
		// to provide flexibility for testing scenarios that don't require full domain validation
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}

		ast.Inspect(file, func(n ast.Node) bool {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok || funcDecl.Name == nil || !strings.HasPrefix(funcDecl.Name.Name, "New") {
				return true
			}

			if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
				if ident, ok := funcDecl.Type.Results.List[0].Type.(*ast.Ident); ok {
					if voTypes[ident.Name] {
						start := fileSet.Position(funcDecl.Pos()).Line
						end := fileSet.Position(funcDecl.End()).Line

						key := funcDecl.Name.Name + ":" + ident.Name
						constructors[key] = &ConstructorInfo{
							File:      path,
							StartLine: start,
							EndLine:   end,
						}
					}
				}
			}

			return true
		})

		return nil
	})

	if err != nil {
		return nil, ge.Pin(err)
	}

	return constructors, nil
}

// isInsideConstructor checks if a given line number is within a constructor function.
//
// Parameters:
//   - file: The file path to check
//   - line: The line number to check
//   - voType: The Value Object type name
//   - constructors: A map of constructor information
//
// Returns:
//   - true if the line is inside a constructor for the specified Value Object, false otherwise
func isInsideConstructor(file string, line int, voType string, constructors map[string]*ConstructorInfo) bool {
	for key, constructor := range constructors {
		if strings.HasSuffix(key, ":"+voType) && constructor.File == file {
			if line >= constructor.StartLine && line <= constructor.EndLine {
				return true
			}
		}
	}

	return false
}

// findZeroValueInitializations scans for zero-value initializations of Value Objects outside constructors.
//
// Parameters:
//   - rootPath: The root directory path to scan for Go files
//   - voTypes: A map of Value Object type names
//   - constructors: A map of constructor information for checking scope
//
// Returns:
//   - A map of violation messages indicating zero-value initialization violations
//   - An error if the scan fails, nil otherwise
func findZeroValueInitializations(rootPath string, voTypes map[string]bool, constructors map[string]*ConstructorInfo) (map[string]bool, error) {
	violations := make(map[string]bool)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || filepath.Ext(path) != ".go" {
			return nil
		}

		// Skip test files - we intentionally allow zero-value initializations in tests
		// to provide flexibility for testing scenarios that don't require full domain validation
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fileSet := token.NewFileSet()
		file, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}

		ast.Inspect(file, func(n ast.Node) bool {
			var compLit *ast.CompositeLit
			var voTypeName string

			if cl, ok := n.(*ast.CompositeLit); ok {
				// Case 1: Direct usage of Location{} (return value, argument, etc.)
				compLit = cl
			} else if assignStmt, ok := n.(*ast.AssignStmt); ok {
				// Case 2: Assignment badLoc := Location{} or badLoc := packageName.Location{}
				for _, rhs := range assignStmt.Rhs {
					if cl, ok := rhs.(*ast.CompositeLit); ok {
						compLit = cl
						break
					}
				}
			} else if returnStmt, ok := n.(*ast.ReturnStmt); ok {
				// Case 3: Return statement return Location{}
				for _, result := range returnStmt.Results {
					if cl, ok := result.(*ast.CompositeLit); ok {
						compLit = cl
						break
					}
				}
			}

			// Skip if no CompositeLit found
			if compLit == nil {
				return true
			}

			// Skip non zero-value initializations
			if len(compLit.Elts) != 0 {
				return true
			}

			// Determine type name
			switch typ := compLit.Type.(type) {
			case *ast.Ident:
				voTypeName = typ.Name
			case *ast.SelectorExpr:
				voTypeName = typ.Sel.Name
			default:
				return true
			}

			// Check if this is a Value Object type
			if !voTypes[voTypeName] {
				return true
			}

			line := fileSet.Position(compLit.Pos()).Line

			// Check if this is inside a constructor
			if !isInsideConstructor(path, line, voTypeName, constructors) {
				violation := fmt.Sprintf("VIOLATION: Direct zero-value initialization of ValueObject %s at %s:%d", voTypeName, path, line)
				violations[violation] = true
			}
			return true
		})

		return nil
	})

	if err != nil {
		return nil, ge.Pin(err)
	}

	return violations, nil
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
	Constructors map[string]*ConstructorInfo
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
	types, err := findValueObjectTypeDeclarations(rootPath)
	if err != nil {
		return nil, ge.Pin(err)
	}

	if len(types) == 0 {
		return nil, nil
	}

	constructors, err := findConstructors(rootPath, types)
	if err != nil {
		return nil, ge.Pin(err)
	}

	violations, err := findZeroValueInitializations(rootPath, types, constructors)
	if err != nil {
		return nil, ge.Pin(err)
	}

	return &ValidateValueObjectsReport{
		Types:        types,
		Constructors: constructors,
		Violations:   violations,
	}, nil
}
