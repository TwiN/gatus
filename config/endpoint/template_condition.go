package endpoint

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"text/template/parse"
	"time"

	"github.com/Masterminds/sprig/v3"

	"github.com/TwiN/gatus/v5/config/gontext"
	"github.com/TwiN/gatus/v5/pattern"
)

// isTemplateCondition reports whether a condition is written using the text/template
// pipeline syntax (e.g. "{{eq .Status 200}}") rather than the legacy bracket DSL.
func isTemplateCondition(condition string) bool {
	trimmed := strings.TrimSpace(condition)
	return strings.HasPrefix(trimmed, "{{") && strings.HasSuffix(trimmed, "}}")
}

// compiledTemplateCondition is the cached, parsed form of a template Condition.
type compiledTemplateCondition struct {
	tmpl       *template.Template
	fields     map[string]bool // top-level Result fields referenced directly, e.g. "Body", "Headers", "IP", "DomainExpiration"
	fieldPaths []string        // distinct dot-joined field paths referenced directly, e.g. "Status", "Body.user.name", in first-appearance order
	parseErr   error
}

// templateConditionCache is keyed by the full condition string, so it stays bounded by the number
// of distinct template conditions across the loaded config (parsed once at startup/reload) for
// typical usage. It has no eviction, so configs that programmatically generate a large and
// ever-changing set of distinct condition strings (e.g. one condition per dynamically discovered
// device) should keep that set bounded to avoid unbounded memory growth.
var templateConditionCache sync.Map // map[string]*compiledTemplateCondition

// getCompiledTemplateCondition parses (or retrieves from cache) the template for a condition.
func getCompiledTemplateCondition(condition string) *compiledTemplateCondition {
	if cached, ok := templateConditionCache.Load(condition); ok {
		return cached.(*compiledTemplateCondition)
	}
	compiled := &compiledTemplateCondition{fields: make(map[string]bool)}
	tmpl, err := template.New("condition").Funcs(templateFuncMap).Parse(condition)
	if err != nil {
		compiled.parseErr = err
	} else {
		compiled.tmpl = tmpl
		paths, hasVariableFieldChain := collectFieldPaths(tmpl.Tree)
		compiled.fieldPaths = paths
		for _, path := range compiled.fieldPaths {
			top := path
			if i := strings.IndexByte(path, '.'); i >= 0 {
				top = path[:i]
			}
			compiled.fields[top] = true
		}
		if hasVariableFieldChain {
			// A variable is being dotted into (e.g. "{{$b := .}}{{if $b.Body}}...{{end}}"), which
			// collectFieldPaths can't resolve back to a field path without full data-flow analysis.
			// Conservatively assume every gated field might be referenced so the corresponding
			// (potentially expensive) reads aren't skipped.
			compiled.fields["Body"] = true
			compiled.fields["Headers"] = true
			compiled.fields["DomainExpiration"] = true
			compiled.fields["IP"] = true
		}
	}
	actual, _ := templateConditionCache.LoadOrStore(condition, compiled)
	return actual.(*compiledTemplateCondition)
}

// collectFieldPaths returns the distinct dot-joined field paths directly referenced by the
// template (e.g. ".Status" -> "Status", ".Body.user.name" -> "Body.user.name"), in
// first-appearance order, along with whether the template dots into a variable (e.g. the
// "$b.Body" in "{{$b := .}}{{if $b.Body}}...{{end}}"). Used both for static gating
// (needsToReadBody, etc.) and for resolving display values for failed/successful conditions.
//
// A variable that aliases a field directly (e.g. "{{$b := .Body}}{{if $b}}...{{end}}") is still
// tracked correctly, since the ".Body" on the right-hand side of the declaration is itself a
// FieldNode. What can't be resolved without full data-flow analysis is a variable that gets
// dotted into after the fact (e.g. "{{$root := .}}{{if $root.Body}}...{{end}}") - the caller is
// expected to treat hasVariableFieldChain conservatively in that case. This also does not resolve
// field accesses chained onto the result of a function call (e.g. the ".id" in
// "(index .Body.data 0).id" is not captured as a path, though ".Body.data" is).
func collectFieldPaths(tree *parse.Tree) (paths []string, hasVariableFieldChain bool) {
	if tree == nil || tree.Root == nil {
		return nil, false
	}
	seen := make(map[string]bool)
	var walk func(n parse.Node)
	walk = func(n parse.Node) {
		if n == nil {
			return
		}
		switch v := n.(type) {
		case *parse.ListNode:
			for _, c := range v.Nodes {
				walk(c)
			}
		case *parse.ActionNode:
			walk(v.Pipe)
		case *parse.PipeNode:
			for _, c := range v.Cmds {
				walk(c)
			}
		case *parse.CommandNode:
			for _, a := range v.Args {
				walk(a)
			}
		case *parse.FieldNode:
			path := strings.Join(v.Ident, ".")
			if path != "" && !seen[path] {
				seen[path] = true
				paths = append(paths, path)
			}
		case *parse.VariableNode:
			if len(v.Ident) > 1 {
				hasVariableFieldChain = true
			}
		case *parse.ChainNode:
			walk(v.Node)
		case *parse.IfNode:
			walk(&v.BranchNode)
		case *parse.RangeNode:
			walk(&v.BranchNode)
		case *parse.WithNode:
			walk(&v.BranchNode)
		case *parse.BranchNode:
			walk(v.Pipe)
			walk(v.List)
			walk(v.ElseList)
		}
	}
	walk(tree.Root)
	return paths, hasVariableFieldChain
}

// templateData is the data made available to a template Condition.
type templateData struct {
	Status                int
	IP                    string
	DNSRCode              string
	ResponseTime          int64 // milliseconds
	Connected             bool
	CertificateExpiration int64 // milliseconds
	DomainExpiration      int64 // milliseconds
	Body                  interface{}
	Headers               map[string]interface{}
	Context               map[string]interface{}
}

func buildTemplateData(result *Result, ctx *gontext.Gontext) *templateData {
	data := &templateData{
		Status:                result.HTTPStatus,
		IP:                    result.IP,
		DNSRCode:              result.DNSRCode,
		ResponseTime:          result.Duration.Milliseconds(),
		Connected:             result.Connected,
		CertificateExpiration: result.CertificateExpiration.Milliseconds(),
		DomainExpiration:      result.DomainExpiration.Milliseconds(),
	}
	if len(result.Body) > 0 {
		var parsed interface{}
		if err := json.Unmarshal(result.Body, &parsed); err == nil {
			data.Body = parsed
		} else {
			data.Body = string(result.Body)
		}
	}
	if len(result.HTTPResponseHeaders) > 0 {
		collapsed := make(map[string]interface{}, len(result.HTTPResponseHeaders))
		for key, values := range result.HTTPResponseHeaders {
			if len(values) == 1 {
				collapsed[key] = values[0]
			} else {
				collapsed[key] = values
			}
		}
		data.Headers = collapsed
	}
	if ctx != nil {
		data.Context = ctx.GetAll()
	}
	return data
}

// evaluateTemplate evaluates a text/template-pipeline Condition.
func (c Condition) evaluateTemplate(result *Result, dontResolveFailedConditions bool, resolveSuccessfulConditions bool, ctx *gontext.Gontext) bool {
	condition := string(c)
	compiled := getCompiledTemplateCondition(condition)
	success := false
	conditionToDisplay := condition
	shouldResolveCondition := func(success bool) bool {
		if success {
			return resolveSuccessfulConditions
		}
		return !dontResolveFailedConditions
	}
	if compiled.parseErr != nil {
		result.AddError(fmt.Sprintf("invalid condition: %s: %s", condition, compiled.parseErr))
	} else {
		data := buildTemplateData(result, ctx)
		var buf strings.Builder
		if err := compiled.tmpl.Execute(&buf, data); err != nil {
			result.AddError(fmt.Sprintf("error evaluating condition %s: %s", condition, err))
		} else {
			switch buf.String() {
			case "true":
				success = true
			case "false":
				success = false
			default:
				result.AddError(fmt.Sprintf("condition %s did not evaluate to a boolean (got %q)", condition, buf.String()))
			}
		}
		if shouldResolveCondition(success) {
			if resolved := formatResolvedFields(compiled.fieldPaths, data); resolved != "" {
				conditionToDisplay = condition + " (" + resolved + ")"
			}
		}
	}
	result.ConditionResults = append(result.ConditionResults, &ConditionResult{Condition: conditionToDisplay, Success: success})
	return success
}

// formatResolvedFields renders "path=value" for each field path that could be resolved against
// data, joined by ", ", in the order the paths first appeared in the template. Paths that can't
// be resolved (e.g. traversal into a non-JSON body, or a missing key) are silently skipped.
func formatResolvedFields(paths []string, data *templateData) string {
	var parts []string
	for _, path := range paths {
		value, ok := resolveFieldPath(data, path)
		if !ok {
			continue
		}
		parts = append(parts, path+"="+formatFieldValue(value))
	}
	return strings.Join(parts, ", ")
}

// resolveFieldPath walks a dot-joined field path (e.g. "Body.user.name") against data, starting
// with a struct field lookup on templateData and falling back to map-key lookups for every
// subsequent segment (since Body/Headers/Context hold arbitrary map[string]interface{} values).
func resolveFieldPath(data *templateData, path string) (interface{}, bool) {
	var current interface{} = data
	for _, segment := range strings.Split(path, ".") {
		v := reflect.ValueOf(current)
		for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
			if v.IsNil() {
				return nil, false
			}
			v = v.Elem()
		}
		switch v.Kind() {
		case reflect.Struct:
			f := v.FieldByName(segment)
			if !f.IsValid() {
				return nil, false
			}
			current = f.Interface()
		case reflect.Map:
			mv := v.MapIndex(reflect.ValueOf(segment))
			if !mv.IsValid() {
				return nil, false
			}
			current = mv.Interface()
		default:
			return nil, false
		}
	}
	return current, true
}

// formatFieldValue renders a resolved field value for display: composite values (maps, slices,
// arrays) are JSON-encoded for readability, everything else uses its default string form. The
// result is truncated at maximumLengthBeforeTruncatingWhenComparedWithPattern characters, mirroring
// the legacy condition path's protection against dumping huge bodies into alert messages.
func formatFieldValue(value interface{}) string {
	var s string
	if str, ok := value.(string); ok {
		s = str
	} else {
		switch reflect.ValueOf(value).Kind() {
		case reflect.Map, reflect.Slice, reflect.Array:
			if b, err := json.Marshal(value); err == nil {
				s = string(b)
			} else {
				s = fmt.Sprint(value)
			}
		default:
			s = fmt.Sprint(value)
		}
	}
	if len(s) > maximumLengthBeforeTruncatingWhenComparedWithPattern {
		return fmt.Sprintf("%.*s...(truncated)", maximumLengthBeforeTruncatingWhenComparedWithPattern, s)
	}
	return s
}

// coerceNumber attempts to interpret v as a number, trying (in order) a duration string
// (converted to milliseconds), an arbitrary-base integer, and a float.
func coerceNumber(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case bool:
		return 0, false
	case string:
		if d, err := time.ParseDuration(n); err == nil {
			return float64(d.Milliseconds()), true
		}
		if i, err := strconv.ParseInt(n, 0, 64); err == nil {
			return float64(i), true
		}
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return f, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func toDisplayString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func templateEq(a, b interface{}) bool {
	an, aok := coerceNumber(a)
	bn, bok := coerceNumber(b)
	if aok && bok {
		return an == bn
	}
	return toDisplayString(a) == toDisplayString(b)
}

func templateNe(a, b interface{}) bool {
	return !templateEq(a, b)
}

func templateLt(a, b interface{}) bool {
	an, _ := coerceNumber(a)
	bn, _ := coerceNumber(b)
	return an < bn
}

func templateLe(a, b interface{}) bool {
	an, _ := coerceNumber(a)
	bn, _ := coerceNumber(b)
	return an <= bn
}

func templateGt(a, b interface{}) bool {
	an, _ := coerceNumber(a)
	bn, _ := coerceNumber(b)
	return an > bn
}

func templateGe(a, b interface{}) bool {
	an, _ := coerceNumber(a)
	bn, _ := coerceNumber(b)
	return an >= bn
}

// templatePat checks whether value matches the given glob pattern.
func templatePat(pat string, value interface{}) bool {
	return pattern.Match(pat, toDisplayString(value))
}

// templateAny checks whether value equals any one of options.
func templateAny(value interface{}, options ...interface{}) bool {
	for _, option := range options {
		if templateEq(value, option) {
			return true
		}
	}
	return false
}

// templateHas checks whether container has the given key, without erroring on a missing key.
func templateHas(container interface{}, key interface{}) bool {
	switch c := container.(type) {
	case map[string]interface{}:
		_, ok := c[toDisplayString(key)]
		return ok
	case []interface{}:
		idx, err := strconv.Atoi(toDisplayString(key))
		return err == nil && idx >= 0 && idx < len(c)
	default:
		return false
	}
}

// conditionFuncMap holds the functions that are central to condition evaluation. These take
// precedence over Sprig's functions of the same name - in practice, that's only "has", which
// Sprig defines as a list-membership check with reversed argument order: has(needle, haystack).
// eq/ne/lt/le/gt/ge/pat/any aren't defined by Sprig (https://masterminds.github.io/sprig/), so
// overriding them here doesn't shadow anything; they're listed for clarity and future-proofing.
var conditionFuncMap = template.FuncMap{
	"eq":  templateEq,
	"ne":  templateNe,
	"lt":  templateLt,
	"le":  templateLe,
	"gt":  templateGt,
	"ge":  templateGe,
	"pat": templatePat,
	"any": templateAny,
	"has": templateHas,
}

// templateFuncMap is the full set of functions available to a template Condition: the Sprig
// function library (https://masterminds.github.io/sprig/) plus conditionFuncMap layered on top.
var templateFuncMap = buildTemplateFuncMap()

func buildTemplateFuncMap() template.FuncMap {
	funcMap := make(template.FuncMap, len(sprig.TxtFuncMap())+len(conditionFuncMap))
	for name, fn := range sprig.TxtFuncMap() {
		funcMap[name] = fn
	}
	for name, fn := range conditionFuncMap {
		funcMap[name] = fn
	}
	return funcMap
}
