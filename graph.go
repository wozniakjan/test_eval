package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var out = flag.String("o", "out_graph.html", "output html")
var in = flag.String("i", "stats.json", "list of input stats.json")

type dataSet struct {
	labels          []string
	backgroundColor string
	data            []string
	stack           string
}
type test struct {
	offset int64
	Name   string  `json:"name"`
	Blocks []block `json:"block"`
}
type block struct {
	Lines     []string `json:"lines"`
	Start     int64    `json:"start"`
	End       int64    `json:"end"`
	BlockType string   `json:"blockType"`
}

func readInput() []test {
	b := make([]test, 0)
	input, e := ioutil.ReadFile(*in)
	if e != nil {
		panic(e)
	}
	json.Unmarshal(input, &b)
	return b //[0:3]
}

func testNames(b []test) string {
	labels := make([]string, 0)
	for _, l := range b {
		labels = append(labels, l.Name)
	}
	return "[\"" + strings.Join(labels, "\",\n \"") + "\"],"
}

func max(tests []test) (int, int64) {
	var max int
	var maxTime int64
	for _, t := range tests {
		if max < len(t.Blocks) {
			max = len(t.Blocks)
		}
		l := t.Blocks[len(t.Blocks)-1]
		if maxTime < l.End {
			maxTime = l.End
		}
	}
	return max, maxTime
}

func toDataSets(tests []test) []dataSet {
	max, _ := max(tests)
	ds := make([]dataSet, max)
	color := "rgba(128,200,128,0.7)"
	for i := 0; i < max; i++ {
		labels := make([]string, len(tests))
		for j := 0; j < len(tests); j++ {
			labels[j] = "[]"
		}
		values := make([]string, len(tests))
		for j := 0; j < len(tests); j++ {
			values[j] = "0"
		}
		ds[i] = dataSet{labels, color, values, "1"}
		if color == "rgba(128,200,128,0.7)" {
			color = "rgba(200,128,128,0.7)"
		} else {
			color = "rgba(128,200,128,0.7)"
		}
	}

	for i, bs := range tests {
		for ib, b := range bs.Blocks {
			labels := make([]string, 0)
			for _, l := range b.Lines {
				labels = append(labels, fmt.Sprintf("%q", trunc(l)))
			}
			for i, j := 0, len(labels)-1; i < j; i, j = i+1, j-1 {
				labels[i], labels[j] = labels[j], labels[i]
			}
			ds[ib].labels[i] = `[` + strings.Join(labels, ", ") + `]`
			ds[ib].data[i] = fmt.Sprintf("%v", b.End-b.Start)
		}
	}
	return ds
}

func trunc(a string) string {
	if len(a) > 100 {
		return a[0:97] + `...`
	}
	return a
}

func dataSets(test []test) string {
	ds := toDataSets(test)
	strs := make([]string, len(ds))
	for i, d := range ds {
		labels := strings.Join(d.labels, ",\n")
		data := strings.Join(d.data, ",\n")
		strs[i] = fmt.Sprintf(
			"{ labels: [%v],\nbackgroundColor: \"%v\",\ndata: [%v], stack: 1 }\n",
			labels,
			d.backgroundColor,
			data)
	}
	return strings.Join(strs, ", ")
}

func renderPage(b []test) {
	f, _ := os.Create(*out)
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()
	data := `
				var data = {
					labels: ` + testNames(b) + `
					datasets: [ ` + dataSets(b) + `]
				};
`
	fmt.Fprintf(w, pre)
	fmt.Fprintf(w, data)
	_, maxTime := max(b)
	fmt.Fprintf(w, post(fmt.Sprintf("%v", int64(float64(maxTime)*1.02))))
}

func main() {
	flag.Parse()
	data := readInput()
	renderPage(data)
}

var pre = `
<!DOCTYPE HTML>
<html>
	<head>  
		<script type="text/javascript">
			window.onload = function () {
				Chart.defaults.groupableBar = Chart.helpers.clone(Chart.defaults.bar);
                Chart.defaults.global.events = ["click"];

				var helpers = Chart.helpers;
				Chart.controllers.groupableBar = Chart.controllers.bar.extend({
					calculateBarX: function (index, datasetIndex) {
						// position the bars based on the stack index
						var stackIndex = this.getMeta().stackIndex;
						return Chart.controllers.bar.prototype.calculateBarX.apply(this, [index, stackIndex]);
					},

					hideOtherStacks: function (datasetIndex) {
						var meta = this.getMeta();
						var stackIndex = meta.stackIndex;

						this.hiddens = [];
						for (var i = 0; i < datasetIndex; i++) {
							var dsMeta = this.chart.getDatasetMeta(i);
							if (dsMeta.stackIndex !== stackIndex) {
								this.hiddens.push(dsMeta.hidden);
								dsMeta.hidden = true;
							}
						}
					},

					unhideOtherStacks: function (datasetIndex) {
						var meta = this.getMeta();
						var stackIndex = meta.stackIndex;

						for (var i = 0; i < datasetIndex; i++) {
							var dsMeta = this.chart.getDatasetMeta(i);
							if (dsMeta.stackIndex !== stackIndex) {
								dsMeta.hidden = this.hiddens.unshift();
							}
						}
					},

					calculateBarY: function (index, datasetIndex) {
						this.hideOtherStacks(datasetIndex);
						var barY = Chart.controllers.bar.prototype.calculateBarY.apply(this, [index, datasetIndex]);
						this.unhideOtherStacks(datasetIndex);
						return barY;
					},

					calculateBarBase: function (datasetIndex, index) {
						this.hideOtherStacks(datasetIndex);
						var barBase = Chart.controllers.bar.prototype.calculateBarBase.apply(this, [datasetIndex, index]);
						this.unhideOtherStacks(datasetIndex);
						return barBase;
					},

					getBarCount: function () {
						var stacks = [];

						// put the stack index in the dataset meta
						Chart.helpers.each(this.chart.data.datasets, function (dataset, datasetIndex) {
							var meta = this.chart.getDatasetMeta(datasetIndex);
							if (meta.bar && this.chart.isDatasetVisible(datasetIndex)) {
								var stackIndex = stacks.indexOf(dataset.stack);
								if (stackIndex === -1) {
									stackIndex = stacks.length;
									stacks.push(dataset.stack);
								}
								meta.stackIndex = stackIndex;
							}
						}, this);

						this.getMeta().stacks = stacks;
						return stacks.length;
					},
				});

`

func post(max string) string {
	return `
				var ctx = document.getElementById("myChart").getContext("2d");
				new Chart(ctx, {
					type: 'groupableBar',
					data: data,
					options: {
						legend: {
							display: false
						},
						scales: {
							yAxes: [{
								ticks: {
									max: ` + max + `,
									beginAtZero: true,
								},
								stacked: true,
							}],
							xAxes: [{
								ticks: {
									display: false,
									beginAtZero: true,
								},
							}]
						},
						tooltips: {
                            enabled: false,
							callbacks: {
								label: function(tooltipItem, data) {
									return data.datasets[tooltipItem.datasetIndex].labels[tooltipItem.index];
								}
							},
                            custom: function(tooltipModel) {
                                var tooltipEl = document.getElementById('chartjs-tooltip');

                                if (!tooltipEl) {
                                    tooltipEl = document.createElement('div');
                                    tooltipEl.id = 'chartjs-tooltip';
                                    tooltipEl.innerHTML = "<table></table>";
                                    document.body.appendChild(tooltipEl);
                                }

                                if (tooltipModel.opacity === 0) {
                                    tooltipEl.style.opacity = 0;
                                    return;
                                }

                                function getBody(bodyItem) {
                                    return bodyItem.lines;
                                }

                                if (tooltipModel.body) {
                                    var titleLines = tooltipModel.title || [];
                                    var bodyLines = tooltipModel.body

                                    var innerHtml = '<thead>';

                                    titleLines.forEach(function(title) {
                                        var link = title.replace(/:/i,"#L");
                                        innerHtml += '<tr><th align="left">' + '<a href="https://github.com/openshift/origin/tree/master'+link+'" style="color:white; text-decoration: none">' + title + '</a>' + '</th></tr>';
                                    });
                                    innerHtml += '</thead><tbody>';

                                    bodyLines.forEach(function(body, i) {
                                        innerHtml += '<tr><td>' + body + '</td></tr>';
                                    });
                                    innerHtml += '</tbody>';

                                    var tableRoot = tooltipEl.querySelector('table');
                                    tableRoot.innerHTML = innerHtml;
                                }

                                var position = this._chart.canvas.getBoundingClientRect();
                                tooltipEl.style.opacity = 0.7;
                                tooltipEl.style.position = 'absolute';
                                tooltipEl.style.left = position.left + tooltipModel.x + 'px';
                                tooltipEl.style.top = position.top + tooltipModel.y + 'px';
                                tooltipEl.style.fontFamily = tooltipModel._bodyFontFamily;
                                tooltipEl.style.fontSize = tooltipModel.bodyFontSize + 'px';
                                tooltipEl.style.fontStyle = tooltipModel._bodyFontStyle;
                                tooltipEl.style.padding = tooltipModel.yPadding + 'px ' + tooltipModel.xPadding + 'px';
                                tooltipEl.style.background = 'black';
                                tooltipEl.style.color = 'white';
                                tooltipEl.style.borderRadius = '5px';
                            }
						}
					}
				});
			}

		</script>
	</head>
	<body>
        <canvas id="myChart"></canvas>
		<script src="./Chart.bundle.js"></script>
	</body>
</html>
`
}
