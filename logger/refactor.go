package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "vendor" || d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || filepath.Base(path) == "refactor.go" {
			return nil
		}

		// Skip the shared/logger and shared/ports directories as they have the definitions
		if strings.Contains(path, "shared"+string(os.PathSeparator)+"logger") ||
			strings.Contains(path, "shared"+string(os.PathSeparator)+"ports") ||
			strings.Contains(path, "shared/logger") ||
			strings.Contains(path, "shared/ports") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}

		changed := false
		ast.Inspect(node, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			methodName := sel.Sel.Name
			if methodName != "Info" && methodName != "Debug" && methodName != "ErrorErr" {
				return true
			}

			if methodName == "Info" || methodName == "Debug" {
				if len(call.Args) >= 2 {
					if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Value == `"system_event"` {
						return true
					}
					// Only insert if it looks like ctx is the first arg
					// Not strictly verifiable, but safe assuming ports.Logger usage
					newArgs := make([]ast.Expr, 0, len(call.Args)+1)
					newArgs = append(newArgs, call.Args[0])
					newArgs = append(newArgs, &ast.BasicLit{Kind: token.STRING, Value: `"system_event"`})
					newArgs = append(newArgs, call.Args[1:]...)
					call.Args = newArgs
					changed = true
				}
			} else if methodName == "ErrorErr" {
				if len(call.Args) >= 3 {
					if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Value == `"system_event"` {
						return true
					}
					newArgs := make([]ast.Expr, 0, len(call.Args)+1)
					newArgs = append(newArgs, call.Args[0])
					newArgs = append(newArgs, &ast.BasicLit{Kind: token.STRING, Value: `"system_event"`})
					newArgs = append(newArgs, call.Args[1:]...)
					call.Args = newArgs
					changed = true
				}
			}

			return true
		})

		if changed {
			var buf bytes.Buffer
			if err := format.Node(&buf, fset, node); err != nil {
				return nil
			}
			os.WriteFile(path, buf.Bytes(), 0644)
			fmt.Println("Updated:", path)
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error:", err)
	}
}
