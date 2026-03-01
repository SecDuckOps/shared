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
	"runtime"
	"strings"
	"sync"
)

var eventMap = map[string]string{
	"start":  "agent_start",
	"init":   "agent_start",
	"sync":   "config_sync_started",
	"fetch":  "config_sync_started",
	"fail":   "operation_failed",
	"error":  "operation_failed",
	"delete": "resource_deleted",
	"create": "resource_created",
	"update": "resource_updated",
	"login":  "auth_login",
}

func inferEvent(msg string) string {
	lower := strings.ToLower(msg)
	for k, v := range eventMap {
		if strings.Contains(lower, k) {
			return v
		}
	}
	return "system_event"
}

func processFile(path string) {
	fset := token.NewFileSet()

	node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return
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

		method := sel.Sel.Name
		if method != "Info" && method != "Debug" && method != "ErrorErr" {
			return true
		}

		if len(call.Args) < 2 {
			return true
		}

		// Handle the case where my previous script injected "system_event"
		isSystemEventInjection := false
		if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING && lit.Value == `"system_event"` {
			isSystemEventInjection = true
		} else if _, ok := call.Args[1].(*ast.BasicLit); ok {
			// If it's a string but NOT "system_event", leave it alone! We customized it in bootstrap etc.
			return true
		}

		event := "system_event"

		// try infer event from message
		var msgLit *ast.BasicLit
		if isSystemEventInjection && len(call.Args) >= 3 {
			// If "system_event" is at Args[1], the message is at Args[2] for Info/Debug, but for ErrorErr it's Args[3]
			msgIdx := 2
			if method == "ErrorErr" {
				// Info(ctx, event, msg)   -> Args[2] is msg
				// ErrorErr(ctx, event, err, msg) -> Args[3] is msg
				if len(call.Args) >= 4 {
					msgIdx = 3
				}
			}
			msgLit, _ = call.Args[msgIdx].(*ast.BasicLit)
		} else {
			// If not injected, it's the old signature: Info(ctx, msg) -> Args[1] is msg, ErrorErr(ctx, err, msg) -> Args[2] is msg
			msgIdx := 1
			if method == "ErrorErr" {
				if len(call.Args) >= 3 {
					msgIdx = 2
				}
			}
			if len(call.Args) > msgIdx {
				msgLit, _ = call.Args[msgIdx].(*ast.BasicLit)
			}
		}

		if msgLit != nil {
			msg := strings.Trim(msgLit.Value, `"`)
			event = inferEvent(msg)
		}

		if isSystemEventInjection {
			// We just replace the existing "system_event" string.
			call.Args[1] = &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf(`"%s"`, event),
			}
			changed = true
			return true
		}

		newArgs := []ast.Expr{
			call.Args[0], // Context
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf(`"%s"`, event),
			},
		}

		newArgs = append(newArgs, call.Args[1:]...) // Append the rest of original args
		call.Args = newArgs
		changed = true

		return true
	})

	if changed {
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, node); err != nil {
			return
		}

		os.WriteFile(path, buf.Bytes(), 0644)
		fmt.Println("Updated:", path)
	}
}

func main() {
	files := make(chan string, 100)
	wg := sync.WaitGroup{}

	workers := runtime.NumCPU()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range files {
				processFile(f)
			}
		}()
	}

	filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			if d.Name() == "vendor" || d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, ".go") &&
			filepath.Base(path) != "refactor.go" && filepath.Base(path) != "refactor2.go" {

			// skip shared logger interface
			if strings.Contains(path, "shared"+string(os.PathSeparator)+"logger") ||
				strings.Contains(path, "shared"+string(os.PathSeparator)+"ports") ||
				strings.Contains(path, "shared/logger") ||
				strings.Contains(path, "shared/ports") {
				return nil
			}

			files <- path
		}

		return nil
	})

	close(files)
	wg.Wait()
}
