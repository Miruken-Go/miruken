package main

import (
    "bytes"
    "flag"
    "fmt"
    "github.com/miruken-go/miruken"
    "go/ast"
    "go/parser"
    "go/token"
    "io"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

var (
    norecursFlag bool   // -norecurs
    stdoutFlag   bool   // -stdout
    outputFlag   string // -output [y.go]
    suffixFlag   string // -suffix [Handler,Provider]
    testsFlag    bool   // -tests
    unexportFlag bool   // -unexported
)

func init() {
    flag.BoolVar(&norecursFlag, "norecurs", false, "skip sub-directories")
    flag.StringVar(&outputFlag, "output", "mirukentypes", "name of the generated .go file")
    flag.StringVar(&suffixFlag, "suffix", "", "suffix of types to emit or * for all")
    flag.BoolVar(&stdoutFlag, "stdout", false, "write to stdout")
    flag.BoolVar(&testsFlag, "tests", false, "generate test types")
    flag.BoolVar(&unexportFlag, "unexported", false, "include unexported names")
}

func main() {
    flag.Parse()

    outFile := strings.TrimSuffix(outputFlag, ".go")

    suffixes := []string{
        "Handler", "Provider", "Consumer", "Receiver",
        "Validator", "Mapper", "Filter", "Service",
    }

    if suffixFlag == "*" {
        suffixes = nil
    } else if strings.HasPrefix(suffixFlag, "+") {
        if suffixFlag != "+" {
            suffixes = append(suffixes, strings.Split(suffixFlag[1:], ",")...)
        }
    } else if suffixFlag != "" {
        suffixes = strings.Split(suffixFlag, ",")
    }

    dir := "."
    if len(flag.Args()) > 0 {
        dir = flag.Args()[0]
    }
    if err :=  parseDir(dir, false, outFile+ ".go", suffixes); err != nil {
        panic(err)
    }
    if testsFlag {
        if err := parseDir(dir, true, outFile+ "_test.go", suffixes); err != nil {
            panic(err)
        }
    }
}

func parseDir(
    dir      string,
    tests    bool,
    outFile  string,
    suffixes []string,
) (err error) {
    dirFile, err := os.Open(dir)
    if err != nil {
        panic(err)
    }
    defer func() {
        err = dirFile.Close()
    }()

    info, err := dirFile.Stat()
    if err != nil {
        panic(err)
    }
    if !info.IsDir() {
        panic("path is not a directory: " + dir)
    }

    var varName string
    if tests {
        varName = "HandlerTestTypes"
    } else {
        varName = "HandlerTypes"
    }

    filter:= func(info os.FileInfo) bool {
        name := info.Name()
        if info.IsDir() {
            return false
        }
        if name == outFile {
            return false
        }
        if filepath.Ext(name) != ".go" {
            return  false
        }
        return strings.HasSuffix(name, "_test.go") == tests
    }

    pkgs, err := parser.ParseDir(token.NewFileSet(), dir, filter, 0)
    if err != nil {
        panic(err)
    }
    for _, pkg := range pkgs {
        var buf bytes.Buffer

        //goland:noinspection GoPrintFunctions
        _, _ = fmt.Fprintln(&buf, "// Code generated by https://github.com/Miruken-Go/miruken/tools/cmd/mirukentypes; DO NOT EDIT.")
        _, _ = fmt.Fprintln(&buf)
        _, _ = fmt.Fprintln(&buf, "package", pkg.Name)
        _, _ = fmt.Fprintln(&buf, "")
        _, _ = fmt.Fprintf(&buf, "import %q\n", mirukenPkgPath)
        _, _ = fmt.Fprintln(&buf, `import "reflect"`)
        _, _ = fmt.Fprintln(&buf, "")

        // Types
        _, _ = fmt.Fprintf(&buf, "var %s = []reflect.Type{\n", varName)
        printTo(&buf, pkg, ast.Typ, "\tmiruken.TypeOf[*%s](),\n", suffixes)
        _, _ = fmt.Fprintln(&buf, "}")
        _, _ = fmt.Fprintln(&buf, "")

        if stdoutFlag {
            if _, err := io.Copy(os.Stdout, &buf); err != nil {
                return err
            }
        } else {
            filename := filepath.Join(dir, outFile)
            newFileData := buf.Bytes()
            oldFileData, _ := os.ReadFile(filename)
            if !bytes.Equal(newFileData, oldFileData) {
                err = os.WriteFile(filename, newFileData, 0660)
                if err != nil {
                    panic(err)
                }
            }
        }
    }

    if !norecursFlag {
        dirs, err := dirFile.Readdir(-1)
        if err != nil {
            panic(err)
        }
        for _, info := range dirs {
            if info.IsDir() {
                if err := parseDir(filepath.Join(dir, info.Name()),
                    tests, outFile, suffixes); err != nil {
                    return err
                }
            }
        }
    }

    return nil
}

func printTo(
    w         io.Writer,
    pkg      *ast.Package,
    kind      ast.ObjKind,
    format    string,
    suffixes  []string,
) {
    var names []string
    for _, f := range pkg.Files {
        for name, object := range f.Scope.Objects {
            if object.Kind == kind && (unexportFlag || ast.IsExported(name)) {
                if spec := object.Decl.(*ast.TypeSpec); spec != nil {
                    if _, ok := spec.Type.(*ast.StructType); ok {
                        if suffixes == nil {
                            names = append(names, name)
                        } else {
                            for _, suffix := range suffixes {
                                if strings.HasSuffix(name, suffix) {
                                    names = append(names, name)
                                    break
                                }
                            }
                        }
                    }
                }
            }
        }
    }
    sort.Strings(names)
    for _, name := range names {
        _, _ = fmt.Fprintf(w, format, name)
    }
}

var mirukenPkgPath = miruken.TypeOf[miruken.Handler]().PkgPath()