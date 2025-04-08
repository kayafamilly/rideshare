module.exports = function(api) {
  api.cache(true);
  return {
    presets: ['babel-preset-expo'],
    plugins: [
      // Plugin for react-native-dotenv
      ["module:react-native-dotenv", {
        "moduleName": "@env", // How you'll import variables (e.g., import { API_URL } from '@env')
        "path": ".env",       // Path to your .env file
        "blacklist": null,    // Optional: variables to exclude
        "whitelist": null,    // Optional: variables to include (if set, only these are included)
        "safe": false,        // Optional: if true, throws error if .env is missing
        "allowUndefined": true // Optional: if false, throws error for undefined variables
      }],
      // Other plugins...
      // The plugin we installed as a dependency (might be needed by react-native-dotenv indirectly)
      // Note: The warning suggested using @babel/plugin-transform-export-namespace-from instead.
      // Let's try the suggested one first. If it fails, we can revert.
      "@babel/plugin-transform-export-namespace-from", // Using the suggested transform plugin
    ]
  };
};