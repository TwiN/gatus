export const helper = {
	methods: {
		generatePrettyTimeAgo(t) {
			let differenceInMs = new Date().getTime() - new Date(t).getTime();
			if (differenceInMs > 3600000) {
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
