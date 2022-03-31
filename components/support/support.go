package support

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	exprv1alpha1 "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

func GetTargetParts(target string) ([]string, error) {
	parts := strings.Split(target, "/")
	switch {
	case len(parts) > 1:
		return parts, nil
	case len(parts) == 1:
		parts = append(parts, parts[0])
		return parts, nil
	default:
		return nil, fmt.Errorf("target malformed")
	}
}

func CompileCELProg(expr string, variables ...string) (cel.Program, error) {
	var varDecls []*exprv1alpha1.Decl
	for _, variable := range variables {
		varDecls = append(varDecls, decls.NewVar(variable, decls.NewMapType(decls.String, decls.Dyn)))
	}
	d := cel.Declarations(varDecls...)
	env, err := cel.NewEnv(d)
	if err != nil {
		return nil, err
	}

	// compile and check for errs
	ast, iss := env.Compile(expr)
	if iss.Err() != nil {
		return nil, iss.Err()
	}

	prog, err := env.Program(ast)
	if err != nil {
		return nil, err
	}
	return prog, nil
}
