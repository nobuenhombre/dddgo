package helpers

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nobuenhombre/suikat/pkg/ge"
)

// IsTypeDeclaration checks if a struct type contains the SomeObject marker field named "_".
//
// Parameters:
//   - file: The AST file to check imports from
//   - structType: The AST struct type to check
//
// Returns:
//   - true if the struct contains the SomeObject marker named "_", false otherwise
type IsTypeDeclaration func(file *ast.File, structType *ast.StructType) bool

// IsSomeObjectTypeDeclaration checks if a struct type contains the SomeObject marker field named "_".
//
// Parameters:
//   - file: The AST file to check imports from
//   - structType: The AST struct type to check
//
// Returns:
//   - true if the struct contains the SomeObject marker named "_", false otherwise
func IsSomeObjectTypeDeclaration(file *ast.File, structType *ast.StructType, fullPackage string, markerField string, declaredName string) bool {
	if structType.Fields == nil {
		return false
	}

	pkgAlias := GetPackageAlias(file, fullPackage)
	if pkgAlias == "" {
		return false
	}

	for _, field := range structType.Fields.List {
		// STRICT CHECK: Only fields explicitly named "_" are considered SomeObject markers
		if len(field.Names) == 1 && field.Names[0].Name == markerField {
			if selector, ok := field.Type.(*ast.SelectorExpr); ok {
				if ident, ok := selector.X.(*ast.Ident); ok {
					if ident.Name == pkgAlias && selector.Sel.Name == declaredName {
						return true
					}
				}
			}
		}
	}

	return false
}

// FindProjectRoot attempts to locate the root directory of the current Go project.
// It traverses up the directory tree starting from the caller's file location
// until it finds a directory containing a go.mod file.
//
// Returns:
//   - The absolute path to the project root directory if found
//   - An empty string if the project root cannot be located
func FindProjectRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return "", ge.New("cannot find project root")
	}

	current := filepath.Dir(filename)

	for {
		_, err := os.Stat(filepath.Join(current, "go.mod"))
		if err == nil {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}

		current = parent
	}

	return "", ge.New("cannot find project root")
}

// GetPackageAlias finds the package alias for a given full package path in the file's imports.
//
// Parameters:
//   - file: The AST file to check imports from
//   - fullPackagePath: The full package path to look for
//
// Returns:
//   - The package alias if found, empty string otherwise
func GetPackageAlias(file *ast.File, fullPackagePath string) string {
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

// FindTypeDeclarations scans the project directory for SomeObject type declarations.
//
// Parameters:
//   - rootPath: The root directory path to scan for Go files
//
// Returns:
//   - A map of SomeObject type names to boolean values indicating their presence
//   - An error if the scan fails, nil otherwise
func FindTypeDeclarations(rootPath string, isTypeDeclaration IsTypeDeclaration) (map[string]bool, error) {
	typeDeclarations := make(map[string]bool)

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

			if isTypeDeclaration(file, structType) {
				typeDeclarations[typeSpec.Name.Name] = true
			}

			return true
		})

		return nil
	})

	if err != nil {
		return nil, ge.Pin(err)
	}

	return typeDeclarations, nil
}

// ConstructorInfo contains location information about a SomeObjects constructor function.
type ConstructorInfo struct {
	File      string
	StartLine int
	EndLine   int
}

// FindConstructors locates all constructor functions for SomeObjects in the project.
//
// Parameters:
//   - rootPath: The root directory path to scan for Go files
//   - voTypes: A map of SomeObjects type names to search constructors for
//
// Returns:
//   - A map of constructor names to their location information
//   - An error if the scan fails, nil otherwise
func FindConstructors(rootPath string, typeDeclarations map[string]bool) (map[string]*ConstructorInfo, error) {
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
					if typeDeclarations[ident.Name] {
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

// IsInsideConstructor checks if a given line number is within a constructor function.
//
// Parameters:
//   - file: The file path to check
//   - line: The line number to check
//   - typeDeclaration: The SomeObject type name
//   - constructors: A map of constructor information
//
// Returns:
//   - true if the line is inside a constructor for the specified SomeObject, false otherwise
func IsInsideConstructor(file string, line int, typeDeclaration string, constructors map[string]*ConstructorInfo) bool {
	for key, constructor := range constructors {
		if strings.HasSuffix(key, ":"+typeDeclaration) && constructor.File == file {
			if line >= constructor.StartLine && line <= constructor.EndLine {
				return true
			}
		}
	}

	return false
}

// FindZeroValueInitializations scans for zero-value initializations of SomeObjects outside constructors.
//
// Parameters:
//   - rootPath: The root directory path to scan for Go files
//   - voTypes: A map of SomeObjects type names
//   - constructors: A map of constructor information for checking scope
//
// Returns:
//   - A map of violation messages indicating zero-value initialization violations
//   - An error if the scan fails, nil otherwise
func FindZeroValueInitializations(rootPath string, markerName string, typeDeclarations map[string]bool, constructors map[string]*ConstructorInfo) (map[string]bool, error) {
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
			var typeName string

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
				typeName = typ.Name
			case *ast.SelectorExpr:
				typeName = typ.Sel.Name
			default:
				return true
			}

			// Check if this is a Value Object type
			if !typeDeclarations[typeName] {
				return true
			}

			line := fileSet.Position(compLit.Pos()).Line

			// Check if this is inside a constructor
			if !IsInsideConstructor(path, line, typeName, constructors) {
				violation := fmt.Sprintf("VIOLATION: Direct zero-value initialization of %s %s at %s:%d", markerName, typeName, path, line)
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
