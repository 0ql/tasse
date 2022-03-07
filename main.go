package main

import (
	"fmt"
	"os"
	"strings"
	"github.com/rjeczalik/notify"
)

type IdGen struct {
	id int
}

func (g *IdGen) genId() string {
	g.id++
	return "id" + fmt.Sprint(g.id)
}

var idGen = &IdGen{
	id: 0,
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type Element struct {
	tagname   string
	id        string
	css       string
	js        string
	innerText string
	children  []*Element
}

func createElement(component string) *Element {
	el := &Element{}
	el.tagname = "div"
	el.id = idGen.genId()
	var css strings.Builder
	var js strings.Builder
	var innerText strings.Builder

	var subComponent strings.Builder

	writeTo := ""
	parentheses := 0 // ()
	braces := 0      // {}
	brackets := 0    // []
	lessThan := 0    // <>

	for _, v := range component {
		switch v {
		case '(':
			parentheses++
		case '{':
			braces++
		case '[':
			brackets++
		case '<':
			lessThan++
		case ')':
			parentheses--
		case '}':
			braces--
		case ']':
			brackets--
		case '>':
			lessThan--
		}

		if parentheses == 1 && braces == 0 && brackets == 0 && lessThan == 0 {
			writeTo = "subComponent"
		} else if parentheses == 0 && braces == 1 && brackets == 0 && lessThan == 0 {
			writeTo = "css"
		} else if parentheses == 0 && braces == 0 && brackets == 1 && lessThan == 0 {
			writeTo = "js"
		} else if parentheses == 0 && braces == 0 && brackets == 0 && lessThan == 1 {
			writeTo = "innerText"
		} else if parentheses == 0 && braces == 0 && brackets == 0 && lessThan == 0 && subComponent.String() != "" {
			str := subComponent.String()
			str = str[1:]
			el.children = append(el.children, createElement(str))
			subComponent.Reset()
			writeTo = ""
		}

		if writeTo == "css" {
			css.WriteRune(v)
		} else if writeTo == "js" {
			js.WriteRune(v)
		} else if writeTo == "innerText" {
			innerText.WriteRune(v)
		} else if writeTo == "subComponent" {
			subComponent.WriteRune(v)
		}
	}
	if css.String() != "" {
		el.css = css.String()[1:len(css.String())-4]
	}
	if js.String() != "" {
		el.js = fmt.Sprintf(`(()=>{let el=document.getElementById("%s");%s;})();`,
			el.id, js.String()[1:len(js.String())-3])
	}
	if innerText.String() != "" {
		el.innerText = innerText.String()[1:len(innerText.String())-4]
	}

	return el
}

func (el *Element) makeHTMLOpen() string {
	return "<" + el.tagname + " id=\"" + el.id + "\" class=\"" + el.css + "\">" + el.innerText
}

func (el *Element) makeHTMLClose() string {
	return "</" + el.tagname + ">"
}

func getBraceContent(braceOpen rune, braceClosed rune, input string) string {
	var sb strings.Builder
	openBraces := 0

	for _, v := range input {
		if v == braceOpen {
			openBraces++
			if openBraces == 1 {
				continue
			}
		} else if v == braceClosed {
			openBraces--
		}
		if openBraces > 0 {
			sb.WriteRune(v)
		}
	}

	return sb.String()
}

var html strings.Builder
var body strings.Builder
var jsBuilder strings.Builder

func evaluator(ast *Element) {
	jsBuilder.WriteString(ast.js)
	body.WriteString(ast.makeHTMLOpen())

	for _, el := range ast.children {
		evaluator(el)
	}

	body.WriteString(ast.makeHTMLClose())
}

func watcher() {
	c := make(chan notify.EventInfo, 1)

	err := notify.Watch("./src/", c, notify.InCloseWrite)
	check(err)

	defer notify.Stop(c)

	for {
		switch ei := <-c; ei.Event() {
		case notify.InCloseWrite:
			fmt.Println(ei.Path(), "changed. Recompiling...")
			compile("./src/example.tasse")
		}
	}
}

func compile(path string) {
	dat, err := os.ReadFile(path)
	check(err)
	str := string(dat)
	idGen.id = 0

	html.Reset()
	body.Reset()
	jsBuilder.Reset()

	// lexer / parser

	root := getBraceContent('(', ')', str)

	ast := createElement(root)
	fmt.Println(ast)

	// evaluator

	jsBuilder.WriteString(`<script>let el;`)

	evaluator(ast)

	jsBuilder.WriteString("</script>")

	html.WriteString(`<html><head><link rel="stylesheet" href="example.css"></head>`)
	html.WriteString("<body>" + body.String() + jsBuilder.String() + "</body></html>")

	err = os.WriteFile("./dist/out.html", []byte(html.String()), 0644)
	check(err)
}

func main() {
	compile("./src/example.tasse")

	watcher()
}
