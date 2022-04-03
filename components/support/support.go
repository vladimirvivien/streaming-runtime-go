package support

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dapr/go-sdk/service/common"
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

func UnmarshalJSON(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ExtractJSONFromInvocation(e *common.InvocationEvent)(map[string]interface{}, error) {
	switch {
	case e.ContentType == "application/json":
		return UnmarshalJSON(e.Data)
	case e.ContentType == "application/cloudevents+json":
		var cloudevent map[string]interface{}
		if err := json.Unmarshal(e.Data, &cloudevent); err != nil {
			return nil, err
		}
		data, ok := cloudevent["data"]
		if !ok{
			fmt.Errorf("cloudevent missing 'data' entry")
		}
		jsonData, ok := data.(map[string]interface{})
		if !ok {
			fmt.Errorf("cloudevent data has unexpected type: %T", data)
		}
		return jsonData, nil
	}
	return nil, fmt.Errorf("unsupported event type: %s", e.ContentType)
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

func SanitizeIdentifier(id string) string {
	if id == "" {
		return id
	}
	return strings.ReplaceAll(strings.ReplaceAll(id, "-", "_"), ".", "_")
}