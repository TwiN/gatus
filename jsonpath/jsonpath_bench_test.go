package jsonpath

import "testing"

func BenchmarkEval(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Eval("ids[0]", []byte(`{"ids": [1, 2]}`))
		Eval("long.simple.walk", []byte(`{"long": {"simple": {"walk": "value"}}}`))
		Eval("data[0].apps[1].name", []byte(`{"data": [{"apps": [{"name":"app1"}, {"name":"app2"}, {"name":"app3"}]}]}`))
	}
}
