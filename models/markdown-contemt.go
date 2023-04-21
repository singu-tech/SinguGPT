package models

import (
    "bytes"
    "github.com/gomarkdown/markdown/ast"
    "github.com/gomarkdown/markdown/parser"
    "io"
)

// 启用的 Markdown 特性
const markdownExtensions = 0 |
    parser.NoIntraEmphasis |
    parser.Tables |
    parser.FencedCode |
    parser.Autolink |
    parser.Strikethrough |
    parser.LaxHTMLBlocks |
    parser.Footnotes |
    parser.NoEmptyLineBeforeBlock |
    parser.HeadingIDs |
    parser.Titleblock |
    parser.AutoHeadingIDs |
    parser.BackslashLineBreak |
    parser.DefinitionLists |
    parser.MathJax |
    parser.Attributes |
    parser.SuperSubscript |
    parser.EmptyLinesBreakList |
    parser.Includes |
    parser.Mmark

// MarkdownContent 文本类型内容
type MarkdownContent struct {
    // 标签
    tag Tag
    // Markdown 语法树
    ast ast.Node
    // Markdown 内容
    markdown string
}

// Type 内容类型
func (t *MarkdownContent) Type() ContentType {
    return ContentTypeMarkdown
}

// Tag 内容标记
func (t *MarkdownContent) Tag() Tag {
    return t.tag
}

// ExtName 适合的文件扩展名
func (t *MarkdownContent) ExtName() string {
    return ".md"
}

// Len 内容长度
func (t *MarkdownContent) Len() int64 {
    return int64(len(t.markdown))
}

// AST Markdown 语法树
func (t *MarkdownContent) AST() ast.Node {
    return t.ast
}

// ToBytes 转为字节数组
func (t *MarkdownContent) ToBytes() []byte {
    return []byte(t.markdown)
}

// ToString 转为字符串
func (t *MarkdownContent) ToString() string {
    return t.markdown
}

// ToReader 转为字节读取流
func (t *MarkdownContent) ToReader() io.Reader {
    return bytes.NewBufferString(t.markdown)
}

// NewMarkdownContent 构造 Markdown 内容
func NewMarkdownContent(tag Tag, markdown string) *MarkdownContent {
    return &MarkdownContent{
        tag:      tag,
        ast:      parser.NewWithExtensions(markdownExtensions).Parse([]byte(markdown)),
        markdown: markdown,
    }
}