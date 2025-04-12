// frontend/contexts/AuthContext.js
import React, { createContext, useState, useEffect, useContext, useMemo } from 'react';
import * as SecureStore from 'expo-secure-store';
import * as Location from 'expo-location';
import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device'; // Added Device import
import { authService } from '../services/api'; // Import the whole service object
import { Alert, Platform } from 'react-native';

// Configure notification handler (runs when notification is received while app is foregrounded)
Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowAlert: true,
    shouldPlaySound: true,
    shouldSetBadge: false,
  }),
});

// Create the context
const AuthContext = createContext(null);

// Create the provider component
export const AuthProvider = ({ children }) => {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(null);
  const [isLoading, setIsLoading] = useState(true);
  const [hasPaymentMethod, setHasPaymentMethodState] = useState(false);

// --- Helper function for Push Notifications ---
async function registerForPushNotificationsAsync() {
  let token;

  if (Device.isDevice) {
    const { status: existingStatus } = await Notifications.getPermissionsAsync();
    let finalStatus = existingStatus;

    if (existingStatus !== 'granted') {
      const { status } = await Notifications.requestPermissionsAsync();
      finalStatus = status;
    }

    if (finalStatus !== 'granted') {
      console.warn('AuthContext: Failed to get push token for push notification! Status:', finalStatus);
      return null; // Return null if permission not granted
    }

    try {
      // Use your own project ID from app.json -> extra.eas.projectId
      const projectId = "05f0284c-69eb-45db-9b05-87374d8b2d68"; // Replace if needed or get dynamically
      token = (await Notifications.getExpoPushTokenAsync({ projectId })).data;
      console.log('AuthContext: Expo Push Token obtained:', token);
    } catch (e) {
        console.error("AuthContext: Failed to get Expo push token", e);
        return null; // Return null on error
    }

  } else {
    console.warn('AuthContext: Must use physical device for Push Notifications');
    return null; // Return null if not a physical device
  }

  // Handle Android notification channel setup here, after getting the token or confirming permission
  if (Platform.OS === 'android' && token) { // Only set channel if we got a token (implies permission granted)
    await Notifications.setNotificationChannelAsync('default', {
      name: 'default',
      importance: Notifications.AndroidImportance.MAX,
      vibrationPattern: [0, 250, 250, 250],
      lightColor: '#FF231F7C',
    });
    console.log("AuthContext: Android Notification Channel 'default' set.");
  }

  return token; // Return the token (or null)
}


  // --- Permission Request and Feature Registration Logic ---
  const requestPermissionsAndRegisterFeatures = async () => {
    console.log("AuthContext: Requesting permissions and registering features...");

    // 1. Location Permission & Update
    try {
      console.log("AuthContext: Checking location permissions...");
      let locationStatus = (await Location.getForegroundPermissionsAsync()).status;
      if (locationStatus !== 'granted') {
        console.log("AuthContext: Location permission not granted, requesting...");
        locationStatus = (await Location.requestForegroundPermissionsAsync()).status;
      }

      if (locationStatus === 'granted') {
        console.log("AuthContext: Location permission granted. Getting current position...");
        let location = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
        console.log("AuthContext: Location obtained:", location.coords);
        // Send location to backend
        await authService.updateUserLocation({ latitude: location.coords.latitude, longitude: location.coords.longitude }); // Call as method
        console.log("AuthContext: Sent location to backend.");
      } else {
        console.log("AuthContext: Location permission denied.");
      }
    } catch (error) {
      console.error("AuthContext: Error handling location permission/update:", error);
      // Don't block flow for location errors
    }

    // 2. Notification Permission & Token Registration using helper function
    try {
      const pushToken = await registerForPushNotificationsAsync(); // Call the helper

      if (pushToken) {
        // Send the token to your backend server
        await authService.registerPushToken(pushToken); // Use the imported service method
        console.log("AuthContext: Sent push token to backend.");
      } else {
        console.log("AuthContext: No push token obtained or sent (permission denied, not a device, or error).");
      }
    } catch (error) {
      // Catch potential errors from registerPushToken or authService.registerPushToken
      console.error("AuthContext: Error during push notification registration or sending:", error);
    }
  };

  // Effect to load token and user data on app start
  useEffect(() => {
    const loadAuthData = async () => {
      console.log("AuthContext: Loading authentication data...");
      setIsLoading(true);
      let userIsAuthenticated = false;
      try {
        const storedToken = await SecureStore.getItemAsync('authToken');
        if (storedToken) {
          console.log('AuthContext: Token found in storage.');
          setToken(storedToken);
          // Set token for subsequent API calls (assuming interceptor picks it up)
          // Note: Interceptor needs to handle token state changes or re-read from store

          console.log("AuthContext: Token loaded, fetching user profile...");
          const profileResponse = await authService.getProfile();
          if (profileResponse.status === 'success' && profileResponse.data) {
            console.log("AuthContext: User profile fetched successfully.");
            setUser(profileResponse.data);
            setHasPaymentMethodState(profileResponse.data.has_payment_method || false);
            userIsAuthenticated = true; // Mark as authenticated for permission request
          } else {
            console.warn("AuthContext: Failed to fetch profile with stored token:", profileResponse.message);
            await SecureStore.deleteItemAsync('authToken');
            setToken(null);
            setUser(null);
            setHasPaymentMethodState(false);
          }
        } else {
          console.log('AuthContext: No token found in storage.');
        }
      } catch (error) {
        console.error('AuthContext: Error during auth data loading or profile fetch:', error);
        // Clear potentially invalid token if profile fetch fails
        await SecureStore.deleteItemAsync('authToken').catch(() => {}); // Ignore error on delete
        setToken(null);
        setUser(null);
        setHasPaymentMethodState(false);
        // Don't show alert here, let login screen handle it
      } finally {
        setIsLoading(false);
        // Request permissions only if user was successfully authenticated during load
        if (userIsAuthenticated) {
           await requestPermissionsAndRegisterFeatures();
        }
      }
    };

    loadAuthData();
  }, []); // Run only once on mount

  // Login function
  const login = async (email, password) => {
    console.log(`AuthContext: Attempting login for ${email}...`);
    setIsLoading(true);
    try {
      const response = await authService.login({ email, password });
      if (response.status === 'success' && response.data?.token && response.data?.user) {
        const { token: newToken, user: userData } = response.data;
        setToken(newToken);
        setUser(userData);
        setHasPaymentMethodState(userData.has_payment_method || false);
        console.log(`AuthContext: Storing token for user ${userData.id}`);
        await SecureStore.setItemAsync('authToken', newToken);

        // Request permissions after successful login
        await requestPermissionsAndRegisterFeatures();

        setIsLoading(false);
        return true;
      } else {
        throw new Error(response.message || 'Login failed: Invalid response from server.');
      }
    } catch (error) {
      console.error('AuthContext: Login error:', error);
      Alert.alert('Login Failed', error.message || 'An unexpected error occurred.');
      setIsLoading(false);
      return false;
    }
  };

  // Signup function
  const signup = async (userData) => {
     console.log(`AuthContext: Attempting signup for ${userData.email}...`);
     setIsLoading(true);
    try {
      const response = await authService.signup(userData);
      if (response.status === 'success' && response.data) {
         console.log(`AuthContext: Signup successful for ${response.data.email}`);
         Alert.alert('Signup Successful', 'You can now log in with your credentials.');
         setIsLoading(false);
         return true;
      } else {
         throw new Error(response.message || 'Signup failed: Invalid response from server.');
      }
    } catch (error) {
      console.error('AuthContext: Signup error:', error);
      Alert.alert('Signup Failed', error.message || 'An unexpected error occurred.');
      setIsLoading(false);
      return false;
    }
  };

  // Logout function
  const logout = async () => {
    console.log('AuthContext: Logging out.');
    setIsLoading(true);
    try {
      // TODO: Optionally unregister push token from backend here
      await SecureStore.deleteItemAsync('authToken');
      setToken(null);
      setUser(null);
      setHasPaymentMethodState(false);
    } catch (error) {
      console.error('AuthContext: Error during logout:', error);
      Alert.alert('Error', 'Failed to log out properly.');
    } finally {
      setIsLoading(false);
    }
  };

  // Function to update user state locally
  const updateUserState = (updatedUserData) => {
      console.log("AuthContext: Updating local user state:", updatedUserData);
      setUser(updatedUserData);
      if (updatedUserData.has_payment_method !== undefined) {
          setHasPaymentMethodState(updatedUserData.has_payment_method);
      }
  };

  // Delete account function
  const deleteAccount = async () => {
      console.log(`AuthContext: Attempting to delete account for user ${user?.id}...`);
      try {
          const response = await authService.deleteAccount(); // Assumes this exists in api.js
          if (response.status === 'success') {
              console.log("AuthContext: Account deleted on backend, logging out locally.");
              await logout();
              return true;
          } else {
              throw new Error(response.message || "Failed to delete account");
          }
      } catch (error) {
          console.error('AuthContext: Delete account error:', error);
          return false;
      }
  };

  // Function to explicitly set the payment method status
  const setHasPaymentMethod = (status) => {
      console.log(`AuthContext: Setting hasPaymentMethod to ${status}`);
      setHasPaymentMethodState(status);
  };

  // Memoize the context value
  const authContextValue = useMemo(
    () => ({
      user,
      token,
      isLoading,
      login,
      logout,
      signup,
      updateUserState,
      deleteAccount,
      hasPaymentMethod,
      setHasPaymentMethod,
    }),
    [user, token, isLoading, hasPaymentMethod]
  );

  // Provide the context value
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