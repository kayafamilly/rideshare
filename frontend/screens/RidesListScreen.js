// frontend/screens/RidesListScreen.js
import React, { useState, useEffect, useCallback } from 'react';
import { View, Text, FlatList, StyleSheet, ActivityIndicator, RefreshControl, TouchableOpacity, Alert } from 'react-native'; // Added Alert
import { useNavigation, useFocusEffect, useRoute } from '@react-navigation/native'; // Added useRoute
import { rideService } from '../services/api'; // Import ride service
import { useAuth } from '../contexts/AuthContext'; // To check login status and potentially payment status

// Component to display a single ride item in the list
const RideItem = ({ item, onSelect, onJoin, currentUserId, joinedRideIds }) => ( // Added joinedRideIds prop
  <View style={styles.rideItemContainer}>
    <TouchableOpacity style={styles.rideItemContent} onPress={() => onSelect(item.id)}>
      <Text style={styles.rideLocation}>{item.start_location} ➔ {item.end_location}</Text>
      {/* Format date and time nicely */}
      <Text style={styles.rideDateTime}>
        {/* Ensure departure_date is treated as a Date object or parse it */}
        {item.departure_date ? new Date(item.departure_date).toLocaleDateString() : 'N/A'} at {item.departure_time ? item.departure_time.substring(0, 5) : 'N/A'}
      </Text>
      {/* Display available seats (assuming API provides 'available_seats') */}
      {/* Display available seats */}
      <Text style={styles.rideSeats}>
        {(item.total_seats ?? 0) - (item.places_taken ?? 0)} seat(s) available
      </Text>
      {/* Add creator info if available later */}
    </TouchableOpacity>
    {/* Add Join Button only if current user is NOT the creator */}
    {/* Add Join Button only if current user is NOT the creator AND has NOT already joined */}
    {currentUserId !== item.user_id && !joinedRideIds.has(item.id) && (
      <TouchableOpacity style={styles.joinButton} onPress={() => onJoin(item.id)} testID={`joinButton-${item.id}`}>
        <Text style={styles.joinButtonText}>Join Ride</Text>
      </TouchableOpacity>
    )}
  </View>
);

// Screen component to display the list of available rides
// This screen now handles both displaying all available rides AND search results
const RidesListScreen = () => {
  const route = useRoute(); // Get route params
  const searchParams = route.params?.searchParams; // Get searchParams if passed
  const navigation = useNavigation();
  const { token, user, hasPaymentMethod: contextHasPaymentMethod } = useAuth(); // Get payment status from context
  const [rides, setRides] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [joinedRideIds, setJoinedRideIds] = useState(new Set()); // Store IDs of joined rides

  // Function to fetch rides (either all available or based on search params)
  const fetchRides = useCallback(async (params = searchParams) => {
    const isSearch = params && Object.keys(params).length > 0;
    console.log(`RidesListScreen: ${isSearch ? `Searching rides with params: ${JSON.stringify(params)}` : 'Fetching all available rides...'}`);

    if (!isRefreshing) setIsLoading(true);
    setError(null);
    try {
      let response;
      if (isSearch) {
        // Call the actual searchRides function from the service
        response = await rideService.searchRides(params);
         if (response.status === 'success' && Array.isArray(response.data)) {
            console.log(`RidesListScreen: Found ${response.data.length} rides matching search.`);
            setRides(response.data);
         } else {
            throw new Error(response.message || 'Failed to search rides: Invalid response format');
         }
      } else {
        response = await rideService.listAvailableRides();
         if (response.status === 'success' && Array.isArray(response.data)) {
            console.log(`RidesListScreen: Found ${response.data.length} available rides.`);
            setRides(response.data);
         } else {
            throw new Error(response.message || 'Failed to fetch rides: Invalid response format');
         }
      }

    } catch (err) {
      console.error('RidesListScreen: Error fetching/searching rides:', err);
      setError(err.message || 'An error occurred while fetching rides.');
    } finally {
      setIsLoading(false);
      setIsRefreshing(false);
    }
  }, [isRefreshing, searchParams]); // Add searchParams dependency

  // Fetch rides when the screen comes into focus
  // Fetch rides when the screen comes into focus or searchParams change
  useFocusEffect(
    useCallback(() => {
      fetchRides(searchParams); // Fetch available/searched rides

      // Also fetch joined ride IDs if user is logged in
      const fetchJoinedIds = async () => {
        if (token && user) {
          try {
            console.log("RidesListScreen: Fetching joined ride IDs...");
            const response = await rideService.listJoinedRides(); // Use existing service
            if (response.status === 'success' && Array.isArray(response.data)) {
              const ids = new Set(response.data.map(ride => ride.id));
              setJoinedRideIds(ids);
              console.log("RidesListScreen: Joined ride IDs fetched:", ids);
            } else {
               console.warn("RidesListScreen: Failed to fetch joined ride IDs:", response.message);
               setJoinedRideIds(new Set()); // Clear on error
            }
          } catch (err) {
             console.error("RidesListScreen: Error fetching joined ride IDs:", err);
             setJoinedRideIds(new Set()); // Clear on error
          }
        } else {
          setJoinedRideIds(new Set()); // Clear if not logged in
        }
      };

      fetchJoinedIds();

    }, [fetchRides, searchParams, token, user]) // Add token/user dependencies
  );

  // Handle pull-to-refresh
  const onRefresh = useCallback(async () => {
    console.log('RidesListScreen: Refreshing rides list...');
    setIsRefreshing(true); // Set refreshing state
    // fetchRides will handle the API call and setting isRefreshing back to false
    await fetchRides(searchParams); // Re-fetch with current params on refresh
  }, [fetchRides]);

  // Handle press on a ride item
  const handleRidePress = (rideId) => { // Navigate to details
    if (!token) {
      // If not logged in, prompt to login/signup
      // Consider navigating to login or showing a modal
      alert('Please log in or sign up to view ride details or join a ride.');
      navigation.navigate('Login'); // Navigate to login screen
      return;
    }
    // Navigate to RideDetailScreen if logged in
    console.log(`RidesListScreen: Navigating to details for ride ${rideId}`);
    navigation.navigate('RideDetail', { rideId });
  };

  // Function to check payment method status using context
  const checkPaymentMethodStatus = () => {
    // Directly use the state from AuthContext
    console.log(`RidesListScreen: Checking payment method status from context: ${contextHasPaymentMethod}`);
    return contextHasPaymentMethod;
  };

  // Handle join button press - Implements payment logic check
  const handleJoinPress = async (rideId) => {
    console.log(`RidesListScreen: Join button pressed for ride: ${rideId}`);
    if (!token) {
      Alert.alert("Login Required", "Please log in or sign up to join a ride.", [{ text: "OK", onPress: () => navigation.navigate('Login') }]);
      return;
    }

    const hasPaymentMethod = checkPaymentMethodStatus(); // No longer async

    if (hasPaymentMethod) {
      // Call backend endpoint for automatic payment
      console.log(`RidesListScreen: User has payment method. Attempting automatic join for ride ${rideId}...`);
      Alert.alert("Processing Payment", "Attempting to join ride using your saved payment method...");
      try {
        const response = await rideService.joinRideAutomatically(rideId);
        if (response.status === 'success') {
          console.log(`RidesListScreen: Automatic join successful for ride ${rideId}.`);
          Alert.alert("Success", response.message || "Successfully joined ride and payment processed.");
          // Optional: Refresh ride list or navigate somewhere else
          onRefresh(); // Refresh the list to potentially update ride status/seats
        } else {
          // This case might not be hit if backend returns non-2xx status for errors
          console.warn(`RidesListScreen: Automatic join API call failed for ride ${rideId}:`, response.message);
          Alert.alert("Join Failed", response.message || "Could not join the ride automatically.");
        }
      } catch (apiError) {
        console.error(`RidesListScreen: Error calling joinRideAutomatically for ride ${rideId}:`, apiError);
        Alert.alert("Error", apiError.message || "An error occurred while trying to join the ride.");
      }
    } else {
      console.log(`RidesListScreen: User does not have payment method. Redirecting to Settings.`);
      // Redirect to Settings to add payment method
      Alert.alert("Payment Method Required", "Please add a payment method in Settings before joining a ride.", [{ text: "Go to Settings", onPress: () => navigation.navigate('Settings') }, { text: "Cancel", style: "cancel" }]);
    }
  };

  // Render loading indicator only on initial load
  if (isLoading && !isRefreshing && !rides.length) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator size="large" color="#007bff" />
        <Text>Loading rides...</Text>
      </View>
    );
  }

  // Render error message
  if (error && !rides.length) { // Show error prominently if list can't be loaded
    return (
      <View style={styles.centered}>
        <Text style={styles.errorText}>Error: {error}</Text>
        <TouchableOpacity onPress={() => fetchRides(searchParams)} style={styles.retryButton}>
           <Text style={styles.retryButtonText}>Retry</Text>
        </TouchableOpacity>
      </View>
    );
  }

  // Render empty list message or the list itself
  return (
    <View style={styles.container}>
      {/* Create Ride button removed, now accessible via tab */}
      <FlatList
        data={rides}
        keyExtractor={(item) => item.id.toString()}
        renderItem={({ item }) => (
          <RideItem item={item} onSelect={handleRidePress} onJoin={handleJoinPress} currentUserId={user?.id} joinedRideIds={joinedRideIds} />
        )}
        style={styles.list}
        contentContainerStyle={styles.listContent}
        refreshControl={
          <RefreshControl refreshing={isRefreshing} onRefresh={onRefresh} colors={["#007bff"]} tintColor={"#007bff"}/>
        }
        // Display message if list is empty after loading/refreshing
        ListEmptyComponent={
            !isLoading && !isRefreshing ? ( // Only show if not loading/refreshing
                 <View style={styles.centeredEmpty}>
                    <Text>{searchParams ? 'No rides found matching your criteria.' : 'No available rides found right now.'}</Text>
                    <Text>{searchParams ? 'Try broadening your search.' : 'Pull down to refresh!'}</Text>
                 </View>
            ) : null // Don't show empty message while loading
        }
      />
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f8f9fa',
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 20,
  },
   centeredEmpty: { // Style for the empty list message
    marginTop: 50, // Add some margin from the top
    alignItems: 'center',
    padding: 20,
  },
  list: {
    flex: 1,
  },
  listContent: {
     padding: 10,
     flexGrow: 1, // Ensure container grows to allow ListEmptyComponent centering
  },
  rideItemContainer: { // Container for content and button
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    backgroundColor: '#ffffff',
    marginBottom: 10,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#eee',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.08, // Slightly reduced opacity
    shadowRadius: 2,
    elevation: 2,
    paddingVertical: 15, // Padding inside the container
    paddingLeft: 15, // Padding for content
  },
  rideItemContent: { // Takes up most space
    flex: 1, // Allow content to take available space before button
  },
  rideLocation: {
    fontSize: 16,
    fontWeight: 'bold',
    marginBottom: 5,
  },
  rideDateTime: {
    fontSize: 14,
    color: '#555',
    marginBottom: 3,
  },
  rideSeats: {
    fontSize: 14,
    color: '#888',
  },
  joinButton: {
    backgroundColor: '#007bff',
    paddingVertical: 8,
    paddingHorizontal: 12,
    borderRadius: 5,
    marginLeft: 10, // Space between content and button
    marginRight: 10, // Padding from the edge
    alignSelf: 'center', // Center vertically within the container
  },
  joinButtonText: {
    color: '#fff',
    fontSize: 14,
    fontWeight: 'bold',
  },
  errorText: {
    color: 'red',
    marginBottom: 10,
    textAlign: 'center',
  },
  retryButton: {
      marginTop: 15,
      backgroundColor: '#007bff',
      paddingVertical: 10,
      paddingHorizontal: 20,
      borderRadius: 5,
  },
  retryButtonText: {
      color: '#fff',
      fontSize: 16,
  },
   // createButton styles removed
});

export default RidesListScreen;