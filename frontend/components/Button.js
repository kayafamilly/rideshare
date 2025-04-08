// frontend/components/Button.js
import React from 'react';
import { TouchableOpacity, Text, StyleSheet, ActivityIndicator } from 'react-native';

// Reusable Button component with loading state
const Button = ({ title, onPress, style, textStyle, loading, disabled, ...props }) => {
  const isDisabled = disabled || loading; // Disable if explicitly disabled or loading

  return (
    <TouchableOpacity
      style={[
        styles.button,
        style,
        isDisabled ? styles.buttonDisabled : null, // Apply disabled style
      ]}
      onPress={onPress}
      disabled={isDisabled} // Use combined disabled state
      {...props}
    >
      {loading ? (
        // Show ActivityIndicator when loading
        <ActivityIndicator size="small" color="#ffffff" />
      ) : (
        // Show button title otherwise
        <Text style={[styles.buttonText, textStyle]}>{title}</Text>
      )}
    </TouchableOpacity>
  );
};

const styles = StyleSheet.create({
  button: {
    backgroundColor: '#007bff', // Example primary color (blue)
    paddingVertical: 15,
    paddingHorizontal: 20,
    borderRadius: 8,
    alignItems: 'center', // Center content horizontally
    justifyContent: 'center', // Center content vertically
    width: '100%', // Take full width by default
    minHeight: 50, // Ensure a minimum height for consistency
  },
  buttonText: {
    color: '#ffffff', // White text
    fontSize: 16,
    fontWeight: 'bold',
  },
  buttonDisabled: {
    backgroundColor: '#aaa', // Grey background when disabled
    opacity: 0.7, // Slightly transparent
  },
});

export default Button;