// frontend/services/api.js
import axios from 'axios';
// Remove Constants import, use react-native-dotenv instead
import { EXPO_PUBLIC_API_URL } from '@env'; // Import directly from @env
import * as SecureStore from 'expo-secure-store'; // For storing/retrieving auth token

// Variable is imported directly via babel plugin
const API_URL = EXPO_PUBLIC_API_URL;
console.log('API Base URL (from @env):', API_URL); // Log for debugging

// Create an axios instance with default configuration
const apiClient = axios.create({
  baseURL: API_URL, // Base URL for all requests
  timeout: 10000, // Request timeout in milliseconds (e.g., 10 seconds)
  headers: {
    'Content-Type': 'application/json', // Default content type
    'Accept': 'application/json',
  },
});

// --- Request Interceptor ---
// Add an interceptor to automatically attach the auth token to requests
apiClient.interceptors.request.use(
  async (config) => {
    // Retrieve the token from secure storage
    const token = await SecureStore.getItemAsync('authToken'); // Key used to store the token
    if (token) {
      // If token exists, add it to the Authorization header
      config.headers.Authorization = `Bearer ${token}`;
      console.log('Attaching token to request:', config.url); // Log token attachment
    } else {
      console.log('No token found for request:', config.url);
    }
    return config; // Return the modified config
  },
  (error) => {
    // Handle request error (e.g., network issue before sending)
    console.error('Axios request error:', error);
    return Promise.reject(error);
  }
);

// --- Response Interceptor ---
// Add an interceptor to handle common responses or errors globally
apiClient.interceptors.response.use(
  (response) => {
    // Any status code within the range of 2xx causes this function to trigger
    // Log successful responses (optional)
    console.log('Axios response success:', response.status, response.config.url);
    // Directly return the data part of the response for convenience
    return response.data;
  },
  (error) => {
    // Any status codes outside the range of 2xx cause this function to trigger
    console.error('Axios response error:', error.response?.status, error.config?.url, error.message);

    // Handle specific error statuses globally if needed
    if (error.response) {
      // The request was made and the server responded with a status code
      // that falls out of the range of 2xx
      console.error('Error data:', error.response.data);
      console.error('Error status:', error.response.status);
      console.error('Error headers:', error.response.headers);

      if (error.response.status === 401) {
        // Handle unauthorized errors (e.g., invalid token)
        console.warn('Unauthorized request - Token might be invalid or expired.');
        // Potential logout logic here:
        // await SecureStore.deleteItemAsync('authToken');
        // Navigate to login screen (requires access to navigation context, tricky here)
        // Maybe set a global state?
      }
      // Return a structured error object or the error response data
      return Promise.reject(error.response.data || { message: error.message });

    } else if (error.request) {
      // The request was made but no response was received
      console.error('Error request:', error.request);
      return Promise.reject({ message: 'Network Error: No response received from server.' });
    } else {
      // Something happened in setting up the request that triggered an Error
      console.error('Error message:', error.message);
      return Promise.reject({ message: error.message });
    }
  }
);


// --- API Service Functions ---

// Authentication Service
export const authService = {
  /**
   * Sends a signup request to the backend.
   * @param {object} userData - User data (email, password, firstName, etc.)
   * @returns {Promise<object>} - Promise resolving with the backend response data
   */
  signup: (userData) => {
    console.log('Calling API: POST /auth/signup');
    return apiClient.post('/auth/signup', userData);
  },

  /**
   * Sends a login request to the backend.
   * @param {object} credentials - User credentials (email, password)
   * @returns {Promise<object>} - Promise resolving with the backend response data (token, user info)
   */
  login: (credentials) => {
    console.log('Calling API: POST /auth/login');
    return apiClient.post('/auth/login', credentials);
  },

  /**
   * Gets the profile of the currently authenticated user. Requires authentication token.
   * @returns {Promise<object>} - Promise resolving with the user profile data
   */
  getProfile: () => {
    console.log('Calling API: GET /users/profile');
    return apiClient.get('/users/profile');
  },
  /**
   * Updates the profile of the currently authenticated user. Requires authentication token.
   * @param {object} profileData - The profile data to update
   * @returns {Promise<object>} - Promise resolving with the updated user profile data
   */
  updateProfile: (profileData) => { // Moved from rideService
    console.log('Calling API: PUT /users/profile');
    return apiClient.put('/users/profile', profileData);
  },
  // Add other auth-related functions here if needed (e.g., logout, password reset)
 
  /**
   * Updates the last known location of the currently authenticated user. Requires authentication token.
   * @param {object} locationData - { latitude, longitude }
   * @returns {Promise<object>} - Promise resolving with the backend response (likely empty on success)
   */
  updateUserLocation: (locationData) => {
  	console.log('Calling API: PUT /users/location');
  	return apiClient.put('/users/location', locationData);
  },
 
  /**
  	* Registers the Expo Push Token with the backend. Requires authentication token.
  	* @param {string} pushToken - The Expo push token.
  	* @returns {Promise<object>} - Promise resolving with the backend response.
  	*/
  registerPushToken: (pushToken) => {
  	console.log('Calling API: POST /users/push-token');
  	// TODO: Implement this endpoint on the backend
  	return apiClient.post('/users/push-token', { token: pushToken });
  },
 };

// Ride Service
export const rideService = {
  /**
   * Creates a new ride. Requires authentication token.
   * @param {object} rideData - { start_location, end_location, departure_date, departure_time, available_seats }
   * @returns {Promise<object>} - Promise resolving with the created ride data
   */
  createRide: (rideData) => {
    console.log('Calling API: POST /rides');
    return apiClient.post('/rides', rideData);
  },

  /**
   * Lists available rides. Public endpoint.
   * @returns {Promise<object>} - Promise resolving with the list of available rides
   */
  listAvailableRides: () => {
    console.log('Calling API: GET /rides');
    return apiClient.get('/rides');
    // TODO: Add params for filtering/pagination later
  },

  /**
   * Gets details for a specific ride. Requires authentication token.
   * @param {string} rideId - The UUID of the ride
   * @returns {Promise<object>} - Promise resolving with the ride details
   */
  getRideDetails: (rideId) => {
    console.log(`Calling API: GET /rides/${rideId}`);
    return apiClient.get(`/rides/${rideId}`);
  },

  /**
   * Allows the authenticated user to join a ride. Requires authentication token.
   * @param {string} rideId - The UUID of the ride to join
   * @returns {Promise<object>} - Promise resolving with the participation details (or payment intent info later)
   */
  joinRide: (rideId) => {
    console.log(`Calling API: POST /rides/${rideId}/join`);
    return apiClient.post(`/rides/${rideId}/join`);
  },

  /**
   * Gets contact information for confirmed participants of a ride. Requires authentication.
   * @param {string} rideId - The UUID of the ride
   * @returns {Promise<object>} - Promise resolving with the list of contacts [{ user_id, first_name, last_name, whatsapp, is_creator }]
   */
   getRideContacts: (rideId) => {
    console.log(`Calling API: GET /rides/${rideId}/contacts`);
    return apiClient.get(`/rides/${rideId}/contacts`);
  },

  /**
   * Searches for available rides based on criteria. Public endpoint.
   * @param {object} searchParams - { start_location?, end_location?, departure_date? }
   * @returns {Promise<object>} - Promise resolving with the list of matching available rides
   */
   searchRides: (searchParams) => {
     console.log(`Calling API: GET /rides/search with params:`, searchParams);
     // Pass searchParams as URL query parameters
     return apiClient.get('/rides/search', { params: searchParams });
   },

   // --- My Rides Functions ---
   listCreatedRides: () => {
     console.log('Calling API: GET /users/me/rides/created');
     return apiClient.get('/users/me/rides/created');
   },
   listJoinedRides: () => {
     console.log('Calling API: GET /users/me/rides/joined');
     return apiClient.get('/users/me/rides/joined');
   },
   listHistoryRides: () => {
     console.log('Calling API: GET /users/me/rides/history');
     return apiClient.get('/users/me/rides/history');
   },
   deleteRide: (rideId) => {
     console.log(`Calling API: DELETE /rides/${rideId}`);
     return apiClient.delete(`/rides/${rideId}`);
   },
   leaveRide: (rideId) => {
     console.log(`Calling API: POST /rides/${rideId}/leave`);
     return apiClient.post(`/rides/${rideId}/leave`);
   },
   /**
    * Attempts to automatically join a ride and charge the saved payment method. Requires authentication.
    * @param {string} rideId - The UUID of the ride to join
    * @returns {Promise<object>} - Promise resolving with success/error message
    */
   joinRideAutomatically: (rideId) => {
     console.log(`Calling API: POST /rides/${rideId}/join-automatic`);
     return apiClient.post(`/rides/${rideId}/join-automatic`);
   },
   /**
    * Gets the participation status of the current user for a specific ride. Requires authentication.
    * @param {string} rideId - The UUID of the ride
    * @returns {Promise<object>} - Promise resolving with { participation_status: string }
    */
   getMyParticipationStatus: (rideId) => {
     console.log(`Calling API: GET /rides/${rideId}/my-status`);
     return apiClient.get(`/rides/${rideId}/my-status`);
   },

   // updateProfile moved to authService

   // --- Settings Functions ---
   deleteAccount: () => {
     console.log('Calling API: DELETE /users/account');
     return apiClient.delete('/users/account');
   },
};

// Payment Service
export const paymentService = {
  /**
   * Creates a payment intent for a specific ride. Requires authentication token.
   * @param {string} rideId - The UUID of the ride to pay for
   * @returns {Promise<object>} - Promise resolving with { client_secret, transaction_id, ... }
   */
  createPaymentIntent: (rideId) => {
    console.log(`Calling API: POST /rides/${rideId}/create-payment-intent`);
    // No request body needed as per current backend implementation
    return apiClient.post(`/rides/${rideId}/create-payment-intent`);
  },

  /**
   * Creates a setup intent to save payment details for future use. Requires authentication token.
   * @returns {Promise<object>} - Promise resolving with { client_secret, customer_id }
   */
  createSetupIntent: () => {
    console.log('Calling API: POST /payments/setup-intent');
    // No request body needed
    return apiClient.post('/payments/setup-intent');
  },
};

// Export the configured axios instance if needed directly elsewhere
export default apiClient;