package gcexporter

import (
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type exporter struct {
	pkg      *types.Package
	imports  map[*types.Package]bool
	toExport []types.Object
	out      io.Writer
}

func Write(pkg *types.Package, out io.Writer, sizes types.Sizes) {
	fmt.Fprintf(out, "package %s\n", pkg.Name())

	e := &exporter{pkg: pkg, imports: make(map[*types.Package]bool), out: out}

	for _, imp := range pkg.Imports() {
		e.addImport(imp)
	}

	for _, name := range pkg.Scope().Names() {
		obj := pkg.Scope().Lookup(name)

		_, isTypeName := obj.(*types.TypeName)
		if obj.IsExported() || isTypeName {
			e.toExport = append(e.toExport, obj)
		}
	}

	for i := 0; i < len(e.toExport); i++ {
		switch o := e.toExport[i].(type) {
		case *types.TypeName:
			fmt.Fprintf(out, "type %s %s\n", e.makeName(o), e.makeType(o.Type().Underlying()))
			if _, isInterface := o.Type().Underlying().(*types.Interface); !isInterface {
				writeMethods := func(methods *types.MethodSet) {
					for i := 0; i < methods.Len(); i++ {
						m := methods.At(i)
						if len(m.Index()) > 1 {
							continue // method of embedded field
						}
						out.Write([]byte("func (? " + e.makeType(m.Recv()) + ") " + e.makeName(m.Obj()) + e.makeSignature(m.Type()) + "\n"))
					}
				}
				writeMethods(o.Type().MethodSet())
				writeMethods(types.NewPointer(o.Type()).MethodSet())
			}
		case *types.Func:
			out.Write([]byte("func " + e.makeName(o) + e.makeSignature(o.Type()) + "\n"))
		case *types.Const:
			optType := ""
			basic, isBasic := o.Type().(*types.Basic)
			if !isBasic || basic.Info()&types.IsUntyped == 0 {
				optType = " " + e.makeType(o.Type())
			}

			basic = o.Type().Underlying().(*types.Basic)
			var val string
			switch {
			case basic.Info()&types.IsBoolean != 0:
				val = strconv.FormatBool(exact.BoolVal(o.Val()))
			case basic.Info()&types.IsInteger != 0:
				if basic.Kind() == types.Uint64 {
					d, _ := exact.Uint64Val(o.Val())
					val = strconv.FormatUint(d, 10)
					break
				}
				d, _ := exact.Int64Val(o.Val())
				val = strconv.FormatInt(d, 10)
			case basic.Info()&types.IsFloat != 0:
				f, _ := exact.Float64Val(o.Val())
				val = strconv.FormatFloat(f, 'b', -1, 64)
			case basic.Info()&types.IsComplex != 0:
				r, _ := exact.Float64Val(exact.Real(o.Val()))
				i, _ := exact.Float64Val(exact.Imag(o.Val()))
				val = fmt.Sprintf("(%s+%si)", strconv.FormatFloat(r, 'b', -1, 64), strconv.FormatFloat(i, 'b', -1, 64))
			case basic.Info()&types.IsString != 0:
				val = fmt.Sprintf("%#v", exact.StringVal(o.Val()))
			default:
				panic("Unhandled constant type: " + basic.String())
			}
			out.Write([]byte("const " + e.makeName(o) + optType + " = " + val + "\n"))
		case *types.Var:
			out.Write([]byte("var " + e.makeName(o) + " " + e.makeType(o.Type()) + "\n"))
		default:
			panic(fmt.Sprintf("Unhandled object: %T\n", o))
		}
	}

	fmt.Fprintf(out, "$$\n")
}

func (e *exporter) addImport(pkg *types.Package) {
	if _, found := e.imports[pkg]; found {
		return
	}
	fmt.Fprintf(e.out, "import %s \"%s\"\n", pkg.Name(), pkg.Path())
	e.imports[pkg] = true
}

func (e *exporter) makeName(o types.Object) string {
	if o.Name() == "" || o.Name() == "_" {
		return "?"
	}
	if o.Pkg() == nil || o.Pkg() == e.pkg {
		return `@"".` + o.Name()
	}
	e.addImport(o.Pkg())
	return `@"` + o.Pkg().Path() + `".` + o.Name()
}

func (e *exporter) makeType(ty types.Type) string {
	switch t := ty.(type) {
	case *types.Basic:
		if t.Kind() == types.UnsafePointer {
			return `@"unsafe".Pointer`
		}
		return t.Name()
	case *types.Array:
		return "[" + strconv.FormatInt(t.Len(), 10) + "]" + e.makeType(t.Elem())
	case *types.Slice:
		return "[]" + e.makeType(t.Elem())
	case *types.Map:
		return "map[" + e.makeType(t.Key()) + "]" + e.makeType(t.Elem())
	case *types.Pointer:
		return "*" + e.makeType(t.Elem())
	case *types.Struct:
		fields := make([]string, t.NumFields())
		for i := range fields {
			field := t.Field(i)
			name := "?"
			if !field.Anonymous() {
				name = e.makeName(field)
			}
			tag := ""
			if t.Tag(i) != "" {
				tag = fmt.Sprintf(" %#v", t.Tag(i))
			}
			fields[i] = name + " " + e.makeType(field.Type()) + tag
		}
		return "struct { " + strings.Join(fields, "; ") + " }"
	case *types.Interface:
		methods := make([]string, t.NumMethods())
		for i := range methods {
			m := t.Method(i)
			methods[i] = e.makeName(m) + e.makeSignature(m.Type())
		}
		return "interface { " + strings.Join(methods, "; ") + " }"
	case *types.Signature:
		return "func " + e.makeSignature(t)
	case *types.Chan:
		switch t.Dir() {
		case types.SendRecv:
			return "chan " + e.makeType(t.Elem())
		case types.SendOnly:
			return "chan<- " + e.makeType(t.Elem())
		case types.RecvOnly:
			return "<-chan " + e.makeType(t.Elem())
		default:
			panic("invalid channel direction")
		}
	case *types.Named:
		if t.Obj().Pkg() == nil {
			return t.Obj().Name()
		}
		found := false
		for _, o := range e.toExport {
			if o == t.Obj() {
				found = true
				break
			}
		}
		if !found {
			e.toExport = append(e.toExport, t.Obj())
		}
		return e.makeName(t.Obj())
	default:
		panic(fmt.Sprintf("Unhandled type: %T\n", t))
	}
}

func (e *exporter) makeSignature(t types.Type) string {
	sig := t.(*types.Signature)
	return "(" + e.makeParameters(sig.Params(), sig.IsVariadic()) + ") (" + e.makeParameters(sig.Results(), false) + ")"
}

func (e *exporter) makeParameters(tuple *types.Tuple, isVariadic bool) string {
	params := make([]string, tuple.Len())
	for i := range params {
		param := tuple.At(i)
		paramType := param.Type()
		dots := ""
		if isVariadic && i == tuple.Len()-1 {
			dots = "..."
			paramType = paramType.(*types.Slice).Elem()
		}
		params[i] = e.makeName(param) + " " + dots + e.makeType(paramType)
	}
	return strings.Join(params, ", ")
}
