// frontend/components/TextInput.js
import React from 'react';
import { TextInput as RNTextInput, StyleSheet, View, Text } from 'react-native';

// Reusable TextInput component with optional label and error message
const TextInput = ({ label, error, style, ...props }) => {
  return (
    <View style={[styles.container, style]}>
      {label && <Text style={styles.label}>{label}</Text>}
      <RNTextInput
        style={[styles.input, error ? styles.inputError : null]}
        placeholderTextColor="#aaa" // Light grey placeholder text
        testID="textInput" // Added testID for testing
        {...props} // Pass down other props like value, onChangeText, placeholder, secureTextEntry etc.
      />
      {error && <Text style={styles.errorText}>{error}</Text>}
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    width: '100%', // Take full width by default
    marginBottom: 15, // Spacing below the input
  },
  label: {
    marginBottom: 5,
    fontSize: 14,
    color: '#333', // Dark grey label
    fontWeight: '500',
  },
  input: {
    borderWidth: 1,
    borderColor: '#ccc', // Light grey border
    borderRadius: 8,
    paddingHorizontal: 15,
    paddingVertical: 12,
    fontSize: 16,
    backgroundColor: '#fff', // White background
  },
  inputError: {
    borderColor: 'red', // Red border for errors
  },
  errorText: {
    marginTop: 4,
    color: 'red',
    fontSize: 12,
  },
});

export default TextInput;