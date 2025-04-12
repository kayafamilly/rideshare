// frontend/screens/LoginScreen.js
import React, { useState } from 'react';
import { View, StyleSheet, Text, Alert, TouchableOpacity, KeyboardAvoidingView, Platform, ScrollView } from 'react-native';
import { useNavigation } from '@react-navigation/native'; // To navigate to SignUp

import TextInput from '../components/TextInput'; // Reusable component
import Button from '../components/Button'; // Reusable component
import { useAuth } from '../contexts/AuthContext'; // Auth context hook

// Screen component for user login
const LoginScreen = () => {
  const navigation = useNavigation();
  const { login, isLoading } = useAuth(); // Get login function and loading state

  // State for form fields
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  // State for form errors (optional for login, backend handles main validation)
  const [errors, setErrors] = useState({});

  // Basic client-side validation (optional)
  const validateForm = () => {
    const newErrors = {};
    if (!email.includes('@')) newErrors.email = 'Please enter a valid email address.';
    if (!password) newErrors.password = 'Password is required.';
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  // Handle login button press
  const handleLogin = async () => {
    if (!validateForm()) {
      // Alert.alert('Validation Error', 'Please check the form fields.'); // Optional client-side alert
      return;
    }

    // Call the login function from AuthContext
    const success = await login(email, password);

    if (success) {
      // Navigation to the main app stack will be handled by the AppNavigator logic
      // based on the token state in AuthContext.
      console.log('Login successful, AuthContext state updated.');
    }
    // Error alert is handled within the login function in AuthContext
  };

  return (
     <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : "height"}
      style={styles.keyboardAvoidingView}
    >
      <ScrollView contentContainerStyle={styles.container}>
        <Text style={styles.title}>Welcome Back!</Text>
        <Text style={styles.subtitle}>Log in to find or share a ride.</Text>

        {/* Form Fields */}
        <TextInput
          label="Email Address"
          value={email}
          onChangeText={setEmail}
          placeholder="your.email@example.com"
          keyboardType="email-address"
          autoCapitalize="none"
          error={errors.email}
          style={styles.inputField}
          testID="loginEmailInput" // Added testID
        />
        <TextInput
          label="Password"
          value={password}
          onChangeText={setPassword}
          placeholder="Enter your password"
          secureTextEntry
          error={errors.password}
          style={styles.inputField}
          testID="loginPasswordInput" // Added testID
        />

        {/* Login Button */}
        <Button
          title="Log In"
          onPress={handleLogin}
          loading={isLoading} // Show loading indicator from context
          style={styles.button}
          testID="loginButton" // Added testID
        />

        {/* Link to SignUp Screen */}
        <TouchableOpacity onPress={() => navigation.navigate('SignUp')} style={styles.signUpLink} testID="navigateToSignUpButton">
          <Text style={styles.signUpLinkText}>Don't have an account? Sign Up</Text>
        </TouchableOpacity>
      </ScrollView>
    </KeyboardAvoidingView>
  );
};

const styles = StyleSheet.create({
   keyboardAvoidingView: {
    flex: 1,
  },
  container: {
    flexGrow: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 20,
    backgroundColor: '#f8f9fa', // Light background color
  },
  title: {
    fontSize: 28,
    fontWeight: 'bold',
    marginBottom: 10,
    color: '#333',
  },
  subtitle: {
    fontSize: 16,
    color: '#666',
    marginBottom: 40, // More space after subtitle
    textAlign: 'center',
  },
  inputField: {
    marginBottom: 20, // Increase spacing between inputs
  },
  button: {
    marginTop: 20, // Add some space above the button
  },
  signUpLink: {
    marginTop: 25, // More space above the signup link
  },
  signUpLinkText: {
    color: '#007bff', // Blue link color
    fontSize: 14,
  },
});

export default LoginScreen;