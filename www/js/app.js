
// Enable range datepicker
$('input[name="daterangepicker"]').daterangepicker();

// Graph stuff
var drawDashboard = function() {
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
            title: title,
            autosize: false,
            width: 300,
            height: 300
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
            title: title,
            autosize: false,
            width: 800,
            height: 300
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
