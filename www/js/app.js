// Tabbed graphs
$('#overview-graphs a').on('click', function (e) {
    e.preventDefault()
    $(this).tab('show')
})

// Graph stuff
var drawDashboard = function() {
    var updateInterval = function(start, end) {
        var from = document.getElementById('date-from')
        var to = document.getElementById('date-to')

        from.value = formatDate(start)
        to.value = formatDate(end)
    }
    
    // Enable range datepicker
    $(function() {
        var start = moment().subtract(29, 'days');
        var end = moment();
    
        function cb(start, end) {
            $('#time-interval-field span').html(start.format('D MMMM') + ' - ' + end.format('D MMMM'));
            updateInterval(start.toDate(), end.toDate())
            updateDashboard() 
        }
    
        $('#time-interval-field').daterangepicker({
            startDate: start,
            endDate: end,
            ranges: {
               'Today': [moment(), moment()],
               'Yesterday': [moment().subtract(1, 'days'), moment().subtract(1, 'days')],
               'Last 7 Days': [moment().subtract(6, 'days'), moment()],
               'Last 30 Days': [moment().subtract(29, 'days'), moment()],
               'This Month': [moment().startOf('month'), moment().endOf('month')],
               'Last Month': [moment().subtract(1, 'month').startOf('month'), moment().subtract(1, 'month').endOf('month')]
            }
        }, cb);
    
        cb(start, end);
    })

    var formatDate = function(d) {
        return d.toISOString().split('T')[0]
    }

    var updateArray = function(dst, src) {
        dst.splice(0, Infinity, ...src)
    }

    var timeIntervalUrlParams = function() {
        return "from=" + document.getElementById('date-from').value + "&to=" + document.getElementById('date-to').value
    }

    // TODO: maybe this is an async function?
    var apiCallGet = function(url) {
        return fetch(url).then(function(res) {
            if (res.ok) {
                return res.json()
            }

            res.text().then(text => console.log("Error requesting method " +
                methodName + ", status:\"" + res.statusText + "\"" +
                ", text: \"" + text + "\""))

            return null
        })
    }

    var fetchGraphDataAsJsonWithTimeInterval = function(methodName) {
        return apiCallGet("api/" + methodName + "?" + timeIntervalUrlParams())
    }

    var updateDonutChart = function(graphName, title) {
        var chartData = [{
            values: [], 
            'marker': {
                'colors': [
                    'rgb(135, 197, 40)',
                    'rgb(255, 92, 111)',
                    'rgb(118, 17, 195)',
                    'rgb(122, 130, 171)',
                ]
            },
            labels: [], 
            type: 'pie', 
            hole: 0.3
        }]
        var layout = {
            width: 340,
            height: 220,
            margin: {
                t: 20,
                l: 20,
                r: 20,
                b: 20
            }
        };

        Plotly.newPlot(graphName, chartData, layout)

        return function() {
            fetchGraphDataAsJsonWithTimeInterval(graphName).then(function(data) {
                var d = data != null ? data.map(v => v["Value"]) : []
                var l = data != null ? data.map(v => v["Key"]) : []
                updateArray(chartData[0].values, d)
                updateArray(chartData[0].labels, l)
                Plotly.redraw(graphName)
            })
        }
    }

    var updateBarChart = function(graphName, title) {
        var chartData = [{
            x: [], 
            y: [], 
            type: 'bar',
            marker: {
                // TODO: find a more elegant solution for this
                color: [
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                    'rgb(149, 205, 234)', 
                ]
            }
        }]
        var layout = {
            width: 660,
            height: 220,
            xaxis: {
                automargin: true,
            },
            yaxis: {
                automargin: true,
            },
            margin: {
                t: 0,
                l: 30,
                r: 0,
                b: 50
            }
        };

        Plotly.newPlot(graphName, chartData, layout)

        return function() {
            fetchGraphDataAsJsonWithTimeInterval(graphName).then(function(data) {
                var x = data != null ? data.map(v => v["Key"]) : []
                var y = data != null ? data.map(v => v["Value"]) : []
                updateArray(chartData[0].x, x)
                updateArray(chartData[0].y, y)
                Plotly.redraw(graphName)
            })
        }
    }

    var updateDeliveryStatus = updateDonutChart("deliveryStatus", "Delivery Status")
    var updateTopBusiestDomainsChart = updateBarChart("topBusiestDomains", "Busiest Domains")
    var updateTopDeferredDomainsChart = updateBarChart("topDeferredDomains", "Most Deferred Domains")
    var updateTopBouncedDomainsChart = updateBarChart("topBouncedDomains", "Most Bounced Domains")

    var updateDashboard = function() {
        updateDeliveryStatus()
        updateTopBusiestDomainsChart()
        updateTopDeferredDomainsChart()
        updateTopBouncedDomainsChart()
    }

    var setupApplicationInfo = function() {
        apiCallGet("/api/appVersion").then(function(data) {
            // TODO: fill UI with version and build info
            // You'll need to "./build.sh release" to have meaningful values
            // Example of received data:
            // {
            //     Commit: "6fbcce9", // empty string on non release version
            //     TagOrBranch: "add-bootstrap", // empty string on non release version
            //     Version: "0.0.0", // "<dev>" on non release version
            // }
            console.log(data)
        })
    }

    setupApplicationInfo()
    updateDashboard()
}
