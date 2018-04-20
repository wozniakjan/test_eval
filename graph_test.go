package main

import (
	"reflect"
	"testing"
)

func TestToDs(t *testing.T) {
	tests := []test{
		test{
			0,
			"test1",
			[]block{
				block{
					[]string{"1line1", "1line2"},
					0,
					7,
					"fast",
				},
				block{
					[]string{"1line3", "1line4"},
					7,
					17,
					"slow",
				},
				block{
					[]string{"1line5", "1line6"},
					17,
					18,
					"fast",
				},
			},
		},
		test{
			0,
			"test2",
			[]block{
				block{
					[]string{"2line1", "2line2"},
					0,
					1,
					"fast",
				},
				block{
					[]string{"2line3", "2line4"},
					1,
					4,
					"slow",
				},
				block{
					[]string{"2line5", "2line6"},
					4,
					6,
					"fast",
				},
				block{
					[]string{"2line7", "2line8"},
					6,
					11,
					"slow",
				},
			},
		},
	}
	expects := []dataSet{
		dataSet{
			[]string{
				"[\"1line2\", \"1line1\"]",
				"[\"2line2\", \"2line1\"]",
			},
			"rgba(128,200,128,0.7)",
			[]string{
				"7",
				"1",
			},
			"1",
		},
		dataSet{
			[]string{
				"[\"1line4\", \"1line3\"]",
				"[\"2line4\", \"2line3\"]",
			},
			"rgba(200,128,128,0.7)",
			[]string{
				"10",
				"3",
			},
			"1",
		},
		dataSet{
			[]string{
				"[\"1line6\", \"1line5\"]",
				"[\"2line6\", \"2line5\"]",
			},
			"rgba(128,200,128,0.7)",
			[]string{
				"1",
				"2",
			},
			"1",
		},
		dataSet{
			[]string{
				"[]",
				"[\"2line8\", \"2line7\"]",
			},
			"rgba(200,128,128,0.7)",
			[]string{
				"0",
				"5",
			},
			"1",
		},
	}
	datasets := toDataSets(tests)
	if len(datasets) != len(expects) {
		t.Errorf("Expected %v ds, got %v", len(expects), len(datasets))
	}
	for i, ds := range datasets {
		if i >= len(expects) {
			t.Errorf("unexpected: %v", ds)
			continue
		}
		if !reflect.DeepEqual(expects[i].labels, ds.labels) {
			t.Errorf("Labels Expected: %v, got %v", expects[i].labels, ds.labels)
		}
		if !reflect.DeepEqual(expects[i].data, ds.data) {
			t.Errorf("Data Expected: %v, got %v", expects[i].data, ds.data)
		}
		if !reflect.DeepEqual(expects[i].backgroundColor, ds.backgroundColor) {
			t.Errorf("Color Expected: %v, got %v", expects[i].backgroundColor, ds.backgroundColor)
		}
	}
}
