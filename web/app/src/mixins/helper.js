export const helper = {
	methods: {
		generatePrettyTimeAgo(t) {
			let differenceInMs = new Date().getTime() - new Date(t).getTime();
			if (differenceInMs > 3*86400000) { // If it was more than 3 days ago, we'll display the number of days ago
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
			return (differenceInMs / 1000).toFixed(0) + " seconds ago";
		},
	}
}
