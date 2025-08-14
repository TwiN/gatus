// Note: The fs.Stats deprecation warning is from Vue CLI's webpack dependencies
// which are not yet compatible with Node.js v23. This is suppressed in the build
// script. All user dependencies have been updated to their latest versions.
// Consider migrating to Vite for better Node.js v23+ compatibility.
module.exports = {
	filenameHashing: false,
	productionSourceMap: false,
	outputDir: '../static',
	publicPath: '/'
}