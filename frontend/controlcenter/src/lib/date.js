import moment from "moment";

export function humanDateTime(d, options) {
  options = Object.assign(
    {
      date: true,
      time: true,
      seconds: true,
      sep: " "
    },
    options
  );

  let format = [];
  if (options.date) format.push("DD MMM. YYYY");
  if (options.time) format.push("h:mmA");

  // TODO: this should be formatted according to the chosen language
  return moment(d).format(format.join(options.sep));
}

// NOTE: we can define shorthand functions here like humanDate, humanTime etc.
