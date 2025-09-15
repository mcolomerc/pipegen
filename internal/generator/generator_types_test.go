package generator

import "testing"

func TestFlinkTypeFromAvroType(t *testing.T) {
	g := &ProjectGenerator{}

	tests := []struct {
		name string
		in   interface{}
		want string
	}{
		{"string", "string", "STRING"},
		{"int", "int", "INT"},
		{"long", "long", "BIGINT"},
		{"float", "float", "FLOAT"},
		{"double", "double", "DOUBLE"},
		{"boolean", "boolean", "BOOLEAN"},
		{"bytes", "bytes", "BYTES"},
		{"unknown-simple", "foobar", "STRING"},

		{"union-null-string", []interface{}{"null", "string"}, "STRING"},
		{"union-null-int", []interface{}{"null", "int"}, "INT"},
		{"union-null-logical-ts", []interface{}{"null", map[string]interface{}{"type": "long", "logicalType": "timestamp-millis"}}, "TIMESTAMP(3)"},

		{"logical-date", map[string]interface{}{"type": "int", "logicalType": "date"}, "DATE"},
		{"logical-timestamp-millis", map[string]interface{}{"type": "long", "logicalType": "timestamp-millis"}, "TIMESTAMP(3)"},
		{"logical-time-micros", map[string]interface{}{"type": "long", "logicalType": "time-micros"}, "TIME(3)"},

		{"complex-array", map[string]interface{}{"type": "array", "items": "string"}, "ARRAY<STRING>"},
		{"complex-map", map[string]interface{}{"type": "map", "values": "string"}, "MAP<STRING, STRING>"},
		{"complex-enum", map[string]interface{}{"type": "enum", "name": "E", "symbols": []interface{}{"A", "B"}}, "STRING"},
		{"complex-record", map[string]interface{}{"type": "record", "name": "R", "fields": []interface{}{}}, "STRING"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := g.flinkTypeFromAvroType(tc.in)
			if got != tc.want {
				t.Fatalf("flinkTypeFromAvroType(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
