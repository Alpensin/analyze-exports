package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type ExportInfo struct {
	File       string
	Line       int
	Name       string
	Type       string
	HasComment bool
}

func main() {
	root := "/path/to/project"

	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory: %v\n", err)
		return
	}

	allExports := make(map[string][]ExportInfo)

	for _, file := range files {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			fmt.Printf("Error parsing %s: %v\n", file, err)
			continue
		}

		exports := analyzeFile(file, fset, node)
		if len(exports) > 0 {
			allExports[file] = exports
		}
	}

	// Вывод результатов
	for file, exports := range allExports {
		fmt.Printf("\nФайл: %s\n", file)
		for i, exp := range exports {
			if !exp.HasComment {
				fmt.Printf("%d. %s (%s) - строка %d\n", i+1, exp.Name, exp.Type, exp.Line)
			}
		}
	}
}

func analyzeFile(filename string, fset *token.FileSet, node *ast.File) []ExportInfo {
	var exports []ExportInfo

	// Собираем все комментарии для быстрого поиска
	commentMap := ast.NewCommentMap(fset, node, node.Comments)

	// Функция для проверки, есть ли комментарий у узла
	hasComment := func(node ast.Node) bool {
		comments := commentMap.Filter(node).Comments()
		return len(comments) > 0
	}

	// Анализ деклараций
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			// Константы, переменные, типы
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					// Константы и переменные
					for _, name := range s.Names {
						if name.IsExported() {
							exportType := "variable"
							if d.Tok == token.CONST {
								exportType = "constant"
							}

							exports = append(exports, ExportInfo{
								File:       filename,
								Line:       fset.Position(name.Pos()).Line,
								Name:       name.Name,
								Type:       exportType,
								HasComment: hasComment(d) || (s.Doc != nil && s.Doc.Text() != ""),
							})
						}
					}
				case *ast.TypeSpec:
					// Типы
					if s.Name.IsExported() {
						typeStr := "type"
						switch s.Type.(type) {
						case *ast.StructType:
							typeStr = "struct"
						case *ast.InterfaceType:
							typeStr = "interface"
						case *ast.FuncType:
							typeStr = "func type"
						}

						exports = append(exports, ExportInfo{
							File:       filename,
							Line:       fset.Position(s.Name.Pos()).Line,
							Name:       s.Name.Name,
							Type:       typeStr,
							HasComment: hasComment(d) || (s.Doc != nil && s.Doc.Text() != ""),
						})
					}
				}
			}
		case *ast.FuncDecl:
			// Функции и методы
			if d.Name.IsExported() {
				exportType := "function"
				if d.Recv != nil && len(d.Recv.List) > 0 {
					exportType = "method"
				}

				exports = append(exports, ExportInfo{
					File:       filename,
					Line:       fset.Position(d.Name.Pos()).Line,
					Name:       d.Name.Name,
					Type:       exportType,
					HasComment: hasComment(d) || (d.Doc != nil && d.Doc.Text() != ""),
				})
			}
		}
	}

	return exports
}
