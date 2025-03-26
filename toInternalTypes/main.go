package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("请提供路径参数")
		return
	}
	path := os.Args[1]

	// 遍历路径，收集所有.go文件
	var fileList []string
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			fileList = append(fileList, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// 处理每个文件
	for _, filePath := range fileList {
		processFile(filePath)
	}
}

func processFile(filePath string) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		log.Printf("解析文件失败 %s: %v", filePath, err)
		return
	}

	// 处理注释
	processComments(file)

	// 处理结构体标签
	processStructTags(file)

	// 将AST写回文件
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		log.Printf("生成代码失败 %s: %v", filePath, err)
		return
	}

	// 写入原文件
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		log.Printf("写入文件失败 %s: %v", filePath, err)
	}
}

func processComments(file *ast.File) {
	// 处理文件顶层的注释组
	var newComments []*ast.CommentGroup
	for _, cg := range file.Comments {
		if !containsKubebuilder(cg) {
			newComments = append(newComments, cg)
		}
	}
	file.Comments = newComments

	// 处理节点关联的注释
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GenDecl:
			if node.Doc != nil && containsKubebuilder(node.Doc) {
				node.Doc = nil
			}
		case *ast.FuncDecl:
			if node.Doc != nil && containsKubebuilder(node.Doc) {
				node.Doc = nil
			}
		case *ast.TypeSpec:
			if node.Doc != nil && containsKubebuilder(node.Doc) {
				node.Doc = nil
			}
		case *ast.Field:
			if node.Doc != nil && containsKubebuilder(node.Doc) {
				node.Doc = nil
			}
		case *ast.ValueSpec:
			if node.Doc != nil && containsKubebuilder(node.Doc) {
				node.Doc = nil
			}
		}
		return true
	})
}

func containsKubebuilder(cg *ast.CommentGroup) bool {
	if cg == nil {
		return false
	}
	for _, comment := range cg.List {
		if strings.Contains(comment.Text, "kubebuilder") {
			return true
		}
	}
	return false
}

func processStructTags(file *ast.File) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.StructType:
			if node.Fields != nil {
				for _, field := range node.Fields.List {
					if field.Tag != nil {
						processFieldTag(field)
					}
				}
			}
		}
		return true
	})
}

func processFieldTag(field *ast.Field) {
	tagVal := field.Tag.Value
	tagStr := strings.Trim(tagVal, "`")
	parts := strings.Fields(tagStr)
	var newParts []string
	for _, part := range parts {
		if !strings.HasPrefix(part, "json:") {
			newParts = append(newParts, part)
		}
	}
	if len(newParts) == 0 {
		field.Tag = nil
	} else {
		field.Tag.Value = "`" + strings.Join(newParts, " ") + "`"
	}
}
