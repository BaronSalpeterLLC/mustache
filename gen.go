// +build ignore

package main

// This program generates mustache_spec_test.go. Invoke it as
// 		go run gen.go -output mustache_spec_test.go

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type JSONSpecDoc struct {
	Tests []JSONSpecTest
}

type JSONSpecTest struct {
	Name        string             `json:"name"`
	Data        interface{}        `json:"data"`
	Expected    string             `json:"expected"`
	Template    string             `json:"template"`
	Description string             `json:"desc"`
	Partials    *map[string]string `json:"partials"`
}

var filename = flag.String("output", "mustache_spec_test.go", "output file name")

var supportedSpecNames = []string{
	"comments",
	"delimiters",
	"interpolation",
	"inverted",
	// "partials",
	"sections",
	// "lambdas",
}

func main() {
	flag.Parse()

	var buf bytes.Buffer

	writeHeader(&buf)

	r := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")

	testSpec := func(scope string, test JSONSpecTest) {
		fmt.Fprint(&buf, `func `)
		fmt.Fprint(&buf, "Test"+strings.Title(scope)+r.Replace(test.Name))
		fmt.Fprintln(&buf, `(t *testing.T) {`)
		if test.Partials != nil {
			for k, v := range *test.Partials {
				name := k + ".mustache"
				fmt.Fprintf(&buf, "\tgeneratePartial(%#v, %#v)\n", name, v)
				fmt.Fprintf(&buf, "\tdefer os.Remove(%#v)\n", name)
			}
		}
		fmt.Fprintln(&buf, `testSpec(t,`)
		fmt.Fprintf(&buf, "%#v,\n%#v,\n%#v", test.Template, test.Expected, test.Data)
		// test.Expected+`, map[string]interface{}{}`+
		fmt.Fprintln(&buf, `)
		}
`)
	}

	generateTests := func(pathName string) {
		data, err := ioutil.ReadFile(pathName)
		if err != nil {
			log.Fatal(err)
		}
		var jsonDocSpec JSONSpecDoc

		err = json.Unmarshal(data, &jsonDocSpec)
		if err != nil {
			log.Fatal(err)
		}

		base := filepath.Base(pathName)
		ext := filepath.Ext(base)

		scope := base[0 : len(base)-len(ext)]

		for _, test := range jsonDocSpec.Tests {
			testSpec(scope, test)
		}
	}

	filepath.Walk("ext/spec/specs/", func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(path, ".json") {
			return nil
		}

		if !isSupportedSpec(path) {
			return nil
		}

		generateTests(path)

		return nil
	})

	data, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("%v\n%s", err, buf.Bytes())
	}
	err = ioutil.WriteFile(*filename, data, 0644)
	if err != nil {
		log.Fatalf("%v\n%v", err, data)
	}
}

func isSupportedSpec(path string) bool {
	for _, specName := range supportedSpecNames {
		if strings.Contains(path, specName) {
			return true
		}
	}
	return false
}

func writeHeader(buf *bytes.Buffer) {
	fmt.Fprintln(buf)
	fmt.Fprintf(buf, "// generated by go run gen.go -output %v; DO NOT EDIT\n", *filename)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, "package mustache")
	fmt.Fprintln(buf)
	fmt.Fprintln(buf)
	fmt.Fprintln(buf, `import (
	"os"
	"strings"
	"testing"
)

func convertHTMLCharsToExpectedFormat(s string) string {
	return strings.Replace(s, "&#34;", "&quot;", -1)
}

func generatePartial(name string, content string) {
	fo, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()
	if _, err := fo.Write([]byte(content)); err != nil {
		panic(err)
	}
}

func testSpec(t *testing.T,
		template string,
		expected string,
		context interface{}) {
		output := convertHTMLCharsToExpectedFormat(Render(template, context))
		if output != expected {
		      t.Errorf("%q\nexpected: %q\nbut got:  %q",
		      	template, expected, output)
		}
}

`)

}
