// frontend/App.js
import React from 'react';
import { StatusBar } from 'expo-status-bar';
import { StripeProvider } from '@stripe/stripe-react-native';
// Remove Constants import, use react-native-dotenv instead
import { EXPO_PUBLIC_STRIPE_PUBLIC_KEY } from '@env'; // Import directly from @env
import { View, Text, StyleSheet } from 'react-native'; // For potential error display

import AppNavigator from './navigation/AppNavigator'; // Import the main navigator
import { AuthProvider } from './contexts/AuthContext'; // Import the Auth Provider

// Variable is imported directly via babel plugin
const stripePublishableKey = EXPO_PUBLIC_STRIPE_PUBLIC_KEY;

// Log the key for debugging (remove in production)
// Log the key imported via react-native-dotenv
console.log('Stripe Publishable Key (from @env):', stripePublishableKey);

export default function App() {
  // Check if the Stripe key is available
  if (!stripePublishableKey) {
    // Log an error and potentially display a message to the user
    console.error("Error: EXPO_PUBLIC_STRIPE_PUBLIC_KEY is not defined. Check .env file and babel config.");
    // Render an error message or a loading state
    return (
      <View style={styles.container}>
        <Text style={styles.errorText}>Error: Stripe configuration is missing.</Text>
        <Text style={styles.errorText}>Please check environment variables.</Text>
        <StatusBar style="auto" />
      </View>
    );
  }

  // Render the app with StripeProvider if the key is available
  return (
    <AuthProvider>
      <StripeProvider
        publishableKey={stripePublishableKey}
        // merchantIdentifier="merchant.com.rideshare" // Optional: Required for Apple Pay setup
        // urlScheme="rideshare" // Optional: Required for handling payment redirects
      >
        {/* AppNavigator contains the main app structure */}
        <AppNavigator />
        {/* StatusBar component for system status bar styling */}
        <StatusBar style="auto" />
      </StripeProvider>
    </AuthProvider>
  );
}

// Styles for the error message display
const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 20,
  },
  errorText: {
    color: 'red',
    textAlign: 'center',
    marginBottom: 10,
  },
});
