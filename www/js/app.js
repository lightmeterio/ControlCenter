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

    var incDate = function(value) {
        // FIXME: That looks freaking ugly, but that's okay for now

        var from = document.getElementById('date-from')
        var to = document.getElementById('date-to')

        var refDate = new Date(from.valueAsDate)

        refDate.setDate(refDate.getDate() + value)
        from.value = formatDate(refDate)

        refDate.setDate(refDate.getDate() + Math.abs(value) - 1)
        to.value = formatDate(refDate)

        updateDashboard()
    }

    var selectNextDay = function() {
        incDate(1)
    }

    var selectPrevDay = function() {
        incDate(-1)
    }

    var selectNextWeek = function() {
        incDate(7)
    }

    var selectPrevWeek = function() {
        incDate(-7)
    }

    var moveToToday = function() {
        var now = new Date()
        document.getElementById('date-from').valueAsDate = now
        document.getElementById('date-to').valueAsDate = now
        updateDashboard()
    }

    var updateArray = function(dst, src) {
        dst.splice(0, Infinity, ...src)
    }

    var timeIntervalUrlParams = function() {
        return "from=" + document.getElementById('date-from').value + "&to=" + document.getElementById('date-to').value
    }

    // TODO: maybe this is an async function?
    var fetchGraphDataAsJsonWithTimeInterval = function(methodName) {
        return fetch("api/" + methodName + "?" + timeIntervalUrlParams()).then(function(res) {
            if (res.ok) {
                return res.json()
            }

            res.text().then(text => console.log("Error requesting method " +
                methodName + ", status:\"" + res.statusText + "\"" +
                ", text: \"" + text + "\""))

            return null
        })
    }

    var updateDonutChart = function(graphName, title) {
        var chartData = [{values: [], labels: [], type: 'pie', hole: 0.4}]
        var layout = {
          margin: {
            t: 30,
            l: 30,
            r: 30,
            b: 30
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
        var chartData = [{x: [], y: [], type: 'bar'}]
        var layout = {
          margin: {
            t: 30,
            l: 30,
            r: 30,
            b: 30
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

    document.getElementById('date-from').onchange = updateDashboard
    document.getElementById('date-to').onchange = updateDashboard
    document.getElementById('prev-date').onclick = selectPrevDay
    document.getElementById('next-date').onclick = selectNextDay
    document.getElementById('prev-week').onclick = selectPrevWeek
    document.getElementById('next-week').onclick = selectNextWeek
    document.getElementById('reload-dashboard').onclick = updateDashboard
    document.getElementById('to-today').onclick = moveToToday

    updateDashboard()
}
