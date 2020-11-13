var insights = document.getElementById("insights")

insights.style.display = "none"

function formatInsightDateTime(d) {
  return moment(d).format('DD MMM YYYY | h:mmA')
}

function formatInsightDescriptionDateTime(d) {
  return moment(d).format('DD MMM. (h:mmA)')
}

var insightTemplate = document.getElementById("insight-template").cloneNode(true)

var allCurrentInsightsData = null

function fetchInsights() {
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

    allCurrentInsightsData = data

    data.forEach(i => {
      var c = insightTemplate.cloneNode(true)
      c.querySelector(".category").innerHTML = buildInsightCategory(i)
      c.querySelector(".time").innerHTML = buildInsightTime(i)
      c.querySelector(".title").innerHTML = buildInsightTitle(i)
      c.querySelector(".rating").classList.add(buildInsightRating(i))
      c.querySelector(".description").innerHTML = buildInsightDescription(i)

      setupInsightInfoIcon(i, c.querySelector(".insight-help-button"))

      insights.appendChild(c)
    })
  })
}

function setupInsightInfoIcon(insight, iconElem) {
  var helpUnavailable = insight.help_link == null

  if (helpUnavailable) {
    iconElem.classList.add("d-none");
    return
  }

  $(iconElem).tooltip()

  iconElem.addEventListener("click", function(event) {
    event.preventDefault()
    _paq.push(['trackEvent', 'InsightsInfoButton', 'click', insight.help_link, insight.ContentType])
    window.open(insight.help_link)
  }, false)
}

function buildInsightTime(insight) {
  return moment(insight.time).format('DD MMM YYYY | h:mmA')
}

function buildInsightCategory(insight) {
  // FIXME We shouldn't capitalise in the code -- leave that for the i18n workflow to decide
  return insight.category.charAt(0).toUpperCase() + insight.category.slice(1)
}

function buildInsightRating(insight) {
  return insight.rating
}

function buildInsightTitle(insight) {
  var s = insightsTitles[insight.content_type]

  if (typeof s == "string") {
    return s
  }

  if (typeof s == "function") {
    return s(insight)
  }

  return "Title for " + insight.content_type
}

function buildInsightDescription(insight) {
  var handler = insightsDescriptions[insight.content_type]

  if (handler == undefined) {
    return "Description for " + insight.content_type
  }

  return handler(insight)
}

// NOTE: yes, this is ugly, and aims to prevent the RBL messages of injecting code in the page.
// It was copied from https://stackoverflow.com/a/9251169/1721672
// Hopefully we'll get rid of all this code when migrating to a proper UI library/framework
function escapeHTML(value) {
  var e = document.createElement("textarea")
  e.textContent = value
  return e.innerHTML
}

function buildInsightRblList(insightId) {
  var insight = allCurrentInsightsData.find(i => i.id == insightId)

  if (insight === undefined) {
    return
  }

  var content = ""

  insight.content.rbls.forEach(r => {
    // FIXME: this needs to made translatable
    content += `\
    <div class="card">\
      <div class="card-body">\
        <h5 class="card-title"><span class="badge badge-pill badge-warning">List</span>` + escapeHTML(r.rbl) + `</h5>\
        <p class="card-text"><span class="message-label">Message:</span>` + escapeHTML(r.text) + `</p>\
      </div>\
    </div>`
  })

  $('#rbl-list-content').html(content)
}

function buildInsightRblCheckedIp(insightId) {
  var insight = allCurrentInsightsData.find(i => i.id == insightId)

  if (insight === undefined) {
    return ""
  }

  return insight.content.address
}

function buildInsightMsgRblDetails(insightId) {
  var insight = allCurrentInsightsData.find(i => i.id == insightId)

  if (insight === undefined) {
    return
  }

  var content = escapeHTML(insight.content.message)

  $('#msg-rbl-list-content').html(content)
}

function buildInsightMsgRblTitle(insightId) {
  var insight = allCurrentInsightsData.find(i => i.id == insightId)

  if (insight === undefined) {
    return ""
  }

  return [insight.content.recipient, insight.content.host]
}

