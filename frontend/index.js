// IMPORTANT: This needs to be the very first import
import 'react-native-gesture-handler';

import { registerRootComponent } from 'expo';
import { GestureHandlerRootView } from 'react-native-gesture-handler';
import { SafeAreaProvider } from 'react-native-safe-area-context'; // Import SafeAreaProvider
import React from 'react';
import { StyleSheet } from 'react-native';

import App from './App';

// registerRootComponent calls AppRegistry.registerComponent('main', () => App);
// It also ensures that whether you load the app in Expo Go or in a native build,
// the environment is set up appropriately
// Wrap the App component with GestureHandlerRootView
const Root = () => (
  <GestureHandlerRootView style={styles.container}>
    <SafeAreaProvider>
      <App />
    </SafeAreaProvider>
  </GestureHandlerRootView>
);

const styles = StyleSheet.create({
  container: { flex: 1 }, // Ensure the root view takes full space
});

registerRootComponent(Root); // Register the wrapped component
