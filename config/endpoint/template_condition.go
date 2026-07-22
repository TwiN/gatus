package endpoint

import (
	"encoding/json"
	"fmt"
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
	tmpl     *template.Template
	fields   map[string]bool // top-level Result fields referenced directly, e.g. "Body", "Headers", "IP", "DomainExpiration"
	parseErr error
}

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
		for _, field := range []string{"Body", "Headers", "IP", "DomainExpiration"} {
			if templateUsesField(tmpl.Tree, field) {
				compiled.fields[field] = true
			}
		}
	}
	actual, _ := templateConditionCache.LoadOrStore(condition, compiled)
	return actual.(*compiledTemplateCondition)
}

// templateUsesField reports whether the parsed template directly references .<field>
// (e.g. .Body, .Headers.Location, (index .Body.data 0).id).
//
// This does not track variable aliases (e.g. "{{$b := .Body}}{{if $b}}...{{end}}").
func templateUsesField(tree *parse.Tree, field string) bool {
	if tree == nil || tree.Root == nil {
		return false
	}
	return walkForField(tree.Root, field)
}

func walkForField(n parse.Node, field string) bool {
	if n == nil {
		return false
	}
	switch v := n.(type) {
	case *parse.ListNode:
		for _, c := range v.Nodes {
			if walkForField(c, field) {
				return true
			}
		}
	case *parse.ActionNode:
		return walkForField(v.Pipe, field)
	case *parse.PipeNode:
		if v == nil {
			return false
		}
		for _, c := range v.Cmds {
			if walkForField(c, field) {
				return true
			}
		}
	case *parse.CommandNode:
		for _, a := range v.Args {
			if walkForField(a, field) {
				return true
			}
		}
	case *parse.FieldNode:
		if len(v.Ident) > 0 && v.Ident[0] == field {
			return true
		}
	case *parse.ChainNode:
		if walkForField(v.Node, field) {
			return true
		}
		if len(v.Field) > 0 && v.Field[0] == field {
			return true
		}
	case *parse.IfNode:
		return walkForField(&v.BranchNode, field)
	case *parse.RangeNode:
		return walkForField(&v.BranchNode, field)
	case *parse.WithNode:
		return walkForField(&v.BranchNode, field)
	case *parse.BranchNode:
		if walkForField(v.Pipe, field) {
			return true
		}
		if walkForField(v.List, field) {
			return true
		}
		if walkForField(v.ElseList, field) {
			return true
		}
	}
	return false
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
func (c Condition) evaluateTemplate(result *Result, ctx *gontext.Gontext) bool {
	condition := string(c)
	compiled := getCompiledTemplateCondition(condition)
	success := false
	if compiled.parseErr != nil {
		result.AddError(fmt.Sprintf("invalid condition: %s: %s", condition, compiled.parseErr))
	} else {
		var buf strings.Builder
		if err := compiled.tmpl.Execute(&buf, buildTemplateData(result, ctx)); err != nil {
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
	}
	result.ConditionResults = append(result.ConditionResults, &ConditionResult{Condition: condition, Success: success})
	return success
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
		if d, err := time.ParseDuration(n); err == nil && d != 0 {
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
// precedence over Sprig's functions of the same name (notably "has", which Sprig defines as a
// list-membership check with reversed argument order: has(needle, haystack)).
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
