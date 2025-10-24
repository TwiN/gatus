/**
 * Generates a human-readable relative time string (e.g., "2 hours ago")
 * @param {string|Date} timestamp - The timestamp to convert
 * @returns {string} Relative time string
 */
export const generatePrettyTimeAgo = (timestamp) => {
  let differenceInMs = new Date().getTime() - new Date(timestamp).getTime();
  if (differenceInMs < 500) {
    return "now";
  }
  if (differenceInMs > 3 * 86400000) { // If it was more than 3 days ago, we'll display the number of days ago
    let days = (differenceInMs / 86400000).toFixed(0);
    return days + " day" + (days !== "1" ? "s" : "") + " ago";
  }
  if (differenceInMs > 3600000) { // If it was more than 1h ago, display the number of hours ago
    let hours = (differenceInMs / 3600000).toFixed(0);
    return hours + " hour" + (hours !== "1" ? "s" : "") + " ago";
  }
  if (differenceInMs > 60000) {
    let minutes = (differenceInMs / 60000).toFixed(0);
    return minutes + " minute" + (minutes !== "1" ? "s" : "") + " ago";
  }
  let seconds = (differenceInMs / 1000).toFixed(0);
  return seconds + " second" + (seconds !== "1" ? "s" : "") + " ago";
}

/**
 * Generates a pretty time difference string between two timestamps
 * @param {string|Date} start - Start timestamp
 * @param {string|Date} end - End timestamp
 * @returns {string} Time difference string
 */
export const generatePrettyTimeDifference = (start, end) => {
  let minutes = Math.ceil((new Date(start) - new Date(end)) / 1000 / 60);
  return minutes + (minutes === 1 ? ' minute' : ' minutes');
}

/**
 * Formats a timestamp into YYYY-MM-DD HH:mm:ss format
 * @param {string|Date} timestamp - The timestamp to format
 * @returns {string} Formatted timestamp
 */
export const prettifyTimestamp = (timestamp) => {
  let date = new Date(timestamp);
  let YYYY = date.getFullYear();
  let MM = ((date.getMonth() + 1) < 10 ? "0" : "") + "" + (date.getMonth() + 1);
  let DD = ((date.getDate()) < 10 ? "0" : "") + "" + (date.getDate());
  let hh = ((date.getHours()) < 10 ? "0" : "") + "" + (date.getHours());
  let mm = ((date.getMinutes()) < 10 ? "0" : "") + "" + (date.getMinutes());
  let ss = ((date.getSeconds()) < 10 ? "0" : "") + "" + (date.getSeconds());
  return YYYY + "-" + MM + "-" + DD + " " + hh + ":" + mm + ":" + ss;
}