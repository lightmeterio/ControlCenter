var insights = document.getElementById("insights")

insights.style.display = "none"

function formatInsightDateTime(d) {
  return moment(d).format('DD MMM YYYY | h:mmA')
}

function formatInsightDescriptionDateTime(d) {
  return moment(d).format('DD MMM. (h:mmA)')
}

function fetchInsights() {
  var insightTemplate = document.getElementById("insight-template")

  var formData = new FormData()

  formData.append("from", selectedDateFrom)
  formData.append("to", selectedDateTo)

  var filter = document.getElementById("insights-filter")

  var setCategoryFilter = function(category) {
      formData.append("filter", "category")
      formData.append("category", category)
  }

  var s = filter.value.split("-")

  if (s.length == 2 && s[0] == "category") {
    setCategoryFilter(s[1])
  }

  formData.append("entries", "6")

  formData.append("order", document.getElementById("insights-sort").value)

  var params = new URLSearchParams(formData)

  fetch("/api/v0/fetchInsights?" + params.toString()).then(res => res.json()).then(data => {
    if (data.length == 0) {
      insights.style.display = "none"
      return
    }

    insights.style.display = "flex"

    while (insights.firstChild) {
      insights.removeChild(insights.firstChild);
    }

    data.forEach(i => {
      var c = insightTemplate.cloneNode(true)
      c.querySelector(".category").innerHTML = buildInsightCategory(i)
      c.querySelector(".time").innerHTML = buildInsightTime(i)
      c.querySelector(".title").innerHTML = buildInsightTitle(i)
      c.querySelector(".rating").classList.add(buildInsightRating(i))
      c.querySelector(".description").innerHTML = buildInsightDescription(i)
      insights.appendChild(c)
    })
  })
}

function buildInsightTime(insight) {
  return moment(insight.Time).format('DD MMM YYYY | h:mmA')
}

function buildInsightCategory(insight) {
  // FIXME We shouldn't capitalise in the code -- leave that for the i18n workflow to decide
  return insight.Category.charAt(0).toUpperCase() + insight.Category.slice(1)
}

function buildInsightRating(insight) {
  return insight.Rating
}

function buildInsightTitle(insight) {
  return insightsTitles[insight.ContentType]
}

function buildInsightDescription(insight) {
  var handler = insightsDescriptions[insight.ContentType]

  if (handler == undefined) {
    return "Description for " + insight.ContentType
  }

  return handler(insight.Content)
}
