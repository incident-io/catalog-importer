package config

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/zyedidia/highlight"
)

func PrettyPrint(jsonnetSource string) {
	h := highlight.NewHighlighter(jsonnetDef)
	matches := h.HighlightString(jsonnetSource)

	lines := strings.Split(jsonnetSource, "\n")
	for lineN, l := range lines {
		colN := 0
		for _, c := range l {
			if group, ok := matches[lineN][colN]; ok {
				// There are more possible groups available than just these ones
				if group == highlight.Groups["statement"] {
					color.Set(color.FgGreen)
				} else if group == highlight.Groups["identifier"] {
					color.Set(color.FgBlue)
				} else if group == highlight.Groups["preproc"] {
					color.Set(color.FgHiRed)
				} else if group == highlight.Groups["special"] {
					color.Set(color.FgRed)
				} else if group == highlight.Groups["constant.string"] {
					color.Set(color.FgCyan)
				} else if group == highlight.Groups["constant"] {
					color.Set(color.FgCyan)
				} else if group == highlight.Groups["constant.specialChar"] {
					color.Set(color.FgHiMagenta)
				} else if group == highlight.Groups["type"] {
					color.Set(color.FgYellow)
				} else if group == highlight.Groups["constant.number"] {
					color.Set(color.FgCyan)
				} else if group == highlight.Groups["comment"] {
					color.Set(color.FgHiGreen)
				} else {
					color.Unset()
				}
			}
			fmt.Print(string(c))
			colN++
		}
		if group, ok := matches[lineN][colN]; ok {
			if group == highlight.Groups["default"] || group == highlight.Groups[""] {
				color.Unset()
			}
		}

		fmt.Print("\n")
	}

	color.Unset()
}

var jsonnetDef *highlight.Def

func init() {
	var err error
	jsonnetDef, err = highlight.ParseDef([]byte(JSONNET_DEF))
	if err != nil {
		panic(fmt.Sprintf("parsing JSONNET_DEF: %v", err))
	}
}

const JSONNET_DEF = `
filetype: jsonnet

detect:
    filename: "\\.jsonnet$"

rules:
    - constant.number: "\\b[-+]?([1-9][0-9]*|0[0-7]*|0x[0-9a-fA-F]+)([uU][lL]?|[lL][uU]?)?\\b"
    - constant.number: "\\b[-+]?([0-9]+\\.[0-9]*|[0-9]*\\.[0-9]+)([EePp][+-]?[0-9]+)?[fFlL]?"
    - constant.number: "\\b[-+]?([0-9]+[EePp][+-]?[0-9]+)[fFlL]?"
    - identifier: "[A-Za-z_][A-Za-z0-9_]*[[:space:]]*[(]"
    - statement: "\\b(break|case|catch|continue|default|delete|do|else|finally)\\b"
    - statement: "\\b(for|function|get|if|in|instanceof|new|return|set|switch)\\b"
    - statement: "\\b(switch|this|throw|try|typeof|var|void|while|with)\\b"
    - constant: "\\b(null|undefined|NaN)\\b"
    - constant: "\\b(true|false)\\b"
    - type: "\\b(Array|Boolean|Date|Enumerator|Error|Function|Math)\\b"
    - type: "\\b(Number|Object|RegExp|String)\\b"
    - statement: "[-+/*=<>!~%?:&|]"
    - constant: "/[^*]([^/]|(\\\\/))*[^\\\\]/[gim]*"
    - constant: "\\\\[0-7][0-7]?[0-7]?|\\\\x[0-9a-fA-F]+|\\\\[bfnrt'\"\\?\\\\]"

    - constant.string:
        start: "\""
        end: "\""
        skip: "\\\\."
        rules:
            - constant.specialChar: "\\\\."

    - constant.string:
        start: "'"
        end: "'"
        skip: "\\\\."
        rules:
            - constant.specialChar: "\\\\."

    - comment:
        start: "//"
        end: "$"
        rules: []

    - comment:
        start: "/\\*"
        end: "\\*/"
        rules: []
`
