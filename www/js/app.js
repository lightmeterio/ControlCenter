// vim: noexpandtab ts=4 sw=4

// Tabbed graphs
$('#overview-graphs a').on('click', function (e) {
    e.preventDefault()
    $(this).tab('show')
})

// TODO: maybe this is an async function?
var apiCallGet = function(url) {
    return fetch(url).then(function(res) {
        if (res.ok) {
            return res.json()
        }

        res.text().then(function(text) {
          console.log("Error requesting url: " +
            url + ", status:\"" + res.statusText + "\"" +
            ", text: \"" + text + "\"")
        })

        return null
    })
}

function setupApplicationInfo(){
    apiCallGet("api/v0/appVersion").then(function(data) {
        if (data != null) {
            let e = document.getElementById("release-info")
            $(e).html(sprintf("<strong>Version:</strong> %s<br/><strong>Commit:</strong> %s<br/><strong>Tag/branch</strong>: %s", data.Version, data.Commit, data.TagOrBranch))
        }
    })
}

var selectedDateFrom = ""
var selectedDateTo = ""

// Graph stuff
var drawDashboard = function() {
    var updateInterval = function(start, end) {
        selectedDateFrom = start
        selectedDateTo = end
        updateDashboard()
        fetchInsights()
    }

    // Enable range datepicker
    $(function() {
        function cb(start, end) {
            $('#time-interval-field span').html(start.format('D MMM') + ' - ' + end.format('D MMM'));
            updateInterval(start.format('YYYY-MM-DD'), end.format('YYYY-MM-DD'))
        }

        var start = moment().subtract(29, 'days');
        var end = moment();

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

    var updateArray = function(dst, src) {
        dst.splice(0, Infinity, ...src)
    }

    var timeIntervalUrlParams = function() {
        return "from=" + selectedDateFrom + "&to=" + selectedDateTo
    }

    var fetchGraphDataAsJsonWithTimeInterval = function(methodName) {
        return apiCallGet("api/v0/" + methodName + "?" + timeIntervalUrlParams())
    }

    var resizers = []

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
            height: 220,
            margin: {
                t: 20,
                l: 20,
                r: 20,
                b: 20
            }
        };

        Plotly.newPlot(graphName, chartData, layout, {responsive: true})

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

        Plotly.newPlot(graphName, chartData, layout, {responsive: true}).then(function() {
            resizers.push(function(dimension) {
                layout.width = dimension.contentRect.width
                Plotly.redraw(graphName)
            })
        })

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

    // Plotly has a bug that makes it unable to resize hidden graphs:
    // https://github.com/plotly/plotly.js/issues/2769
    // We try to workaround it
    var setupResizers = function() {
        // Bail out, no support for ResizeObserver
        if (window.ResizeObserver === undefined) {
            return function() {}
        }

        var graphAreaResizeObserver = new ResizeObserver(function(entry) {
                for (cb in resizers) {
                    resizers[cb](entry[0])
                }
        })

        return function(e) {
            graphAreaResizeObserver.observe(e)
        }
    }()

    setupResizers(document.getElementById('basic-graphs-area'))

    setupApplicationInfo()
}

// enable tooltips everywhere
$(function () {
  $('[data-toggle="tooltip"]').tooltip()
})

// enable keyboard 'enter' to submit form
$(function(){
    $('.form-control').keypress(function(e){
        if(e.which == 13) {
            $( "#submit" ).click()
        }
    })
})

// password field show / hide
$(document).ready(function() {
    $("#show_hide_password a").on('click', function(event) {
        event.preventDefault();
        if($('#show_hide_password input').attr("type") == "text"){
            $('#show_hide_password input').attr('type', 'password');
            $('#show_hide_password i').addClass( "fa-eye-slash" );
            $('#show_hide_password i').removeClass( "fa-eye" );
        }else if($('#show_hide_password input').attr("type") == "password"){
            $('#show_hide_password input').attr('type', 'text');
            $('#show_hide_password i').removeClass( "fa-eye-slash" );
            $('#show_hide_password i').addClass( "fa-eye" );
        }
    });
});

// create welcome text for homepage
$(document).ready(function() {
    if($(".greeting h3").length){
        $(".greeting h3").html(greetingText());
    }
});
