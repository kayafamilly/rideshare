// frontend/contexts/AuthContext.js
import React, { createContext, useState, useEffect, useContext, useMemo } from 'react';
import * as SecureStore from 'expo-secure-store';
import { authService } from '../services/api'; // Import the auth service functions
import { Alert } from 'react-native'; // For showing simple alerts

// Create the context
const AuthContext = createContext(null);

// Create the provider component
export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null); // Store user data
  const [token, setToken] = useState(null); // Store auth token
  const [isLoading, setIsLoading] = useState(true); // Loading state for initial check

  // Effect to load token from storage on app start
  useEffect(() => {
    const loadAuthData = async () => {
      setIsLoading(true);
      try {
        const storedToken = await SecureStore.getItemAsync('authToken');
        if (storedToken) {
          console.log('AuthContext: Token found in storage.');
          setToken(storedToken);
          // TODO: Optionally fetch user profile here using the token
          // For now, we assume token presence means logged in, user data comes from login/signup
        } else {
          console.log('AuthContext: No token found in storage.');
        }
      } catch (error) {
        console.error('AuthContext: Error loading auth token:', error);
        Alert.alert('Error', 'Failed to load authentication state.');
      } finally {
        setIsLoading(false);
      }
    };

    loadAuthData();
  }, []); // Run only once on mount

  // Login function
  const login = async (email, password) => {
    setIsLoading(true);
    try {
      const response = await authService.login({ email, password });
      console.log('AuthContext: Login successful:', response.data); // Backend response structure might be { status: 'success', data: { token: '...', user: {...} } }
      if (response.status === 'success' && response.data?.token && response.data?.user) {
        const { token: newToken, user: userData } = response.data;
        setToken(newToken);
        setUser(userData);
        await SecureStore.setItemAsync('authToken', newToken); // Store token securely
        setIsLoading(false);
        return true; // Indicate success
      } else {
        throw new Error(response.message || 'Login failed: Invalid response from server.');
      }
    } catch (error) {
      console.error('AuthContext: Login error:', error);
      const errorMessage = error.message || 'An unexpected error occurred during login.';
      Alert.alert('Login Failed', errorMessage);
      setIsLoading(false);
      return false; // Indicate failure
    }
  };

  // Signup function
  const signup = async (userData) => {
     setIsLoading(true);
    try {
      // Call the signup API endpoint
      const response = await authService.signup(userData);
      console.log('AuthContext: Signup response:', response); // Expected: { status: 'success', message: '...', data: user }
      if (response.status === 'success' && response.data) {
         // Optionally log the user in directly after signup
         // For now, just show success and let them log in separately
         Alert.alert('Signup Successful', 'You can now log in with your credentials.');
         setIsLoading(false);
         return true; // Indicate success
      } else {
         throw new Error(response.message || 'Signup failed: Invalid response from server.');
      }
    } catch (error) {
      console.error('AuthContext: Signup error:', error);
      const errorMessage = error.message || 'An unexpected error occurred during signup.';
      Alert.alert('Signup Failed', errorMessage);
      setIsLoading(false);
      return false; // Indicate failure
    }
  };

  // Logout function
  const logout = async () => {
    console.log('AuthContext: Logging out.');
    setIsLoading(true);
    try {
      await SecureStore.deleteItemAsync('authToken'); // Remove token from storage
      setToken(null);
      setUser(null);
      // TODO: Optionally call a backend logout endpoint if it exists
    } catch (error) {
      console.error('AuthContext: Error during logout:', error);
      Alert.alert('Error', 'Failed to log out properly.');
    } finally {
      setIsLoading(false);
    }
  };

  // Function to update user state (e.g., after profile update)
  // Note: This only updates the local state, assumes backend call was successful
  const updateUserState = (updatedUserData) => {
      setUser(updatedUserData);
  };

  // Delete account function
  const deleteAccount = async () => {
      // Note: Confirmation Alert is handled in SettingsScreen
      try {
          const response = await authService.deleteAccount();
          if (response.status === 'success') {
              // If backend confirms deletion, log out locally
              await logout(); // Clear token and user state
              return true; // Indicate success
          } else {
              throw new Error(response.message || "Failed to delete account");
          }
      } catch (error) {
          console.error('AuthContext: Delete account error:', error);
          // Alert is handled in SettingsScreen, just return failure
          return false; // Indicate failure
      }
  };

  // Memoize the context value to prevent unnecessary re-renders
  const authContextValue = useMemo(
    () => ({
      user,
      token,
      isLoading,
      login,
      logout,
      signup,
      updateUserState, // Expose function to update user state
      deleteAccount,   // Expose delete account function
    }),
    [user, token, isLoading] // Dependencies remain the same as functions are stable refs
  );

  // Provide the context value to children components
  return (
    <AuthContext.Provider value={authContextValue}>
      {children}
    </AuthContext.Provider>
  );
};

// Custom hook to use the AuthContext
export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};