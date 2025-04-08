// frontend/screens/RidesListScreen.js
import React, { useState, useEffect, useCallback } from 'react';
import { View, Text, FlatList, StyleSheet, ActivityIndicator, RefreshControl, TouchableOpacity } from 'react-native';
import { useNavigation, useFocusEffect, useRoute } from '@react-navigation/native'; // Added useRoute
import { rideService } from '../services/api'; // Import ride service (will add searchRides later)
import { useAuth } from '../contexts/AuthContext'; // To check login status for conditional UI

// Component to display a single ride item in the list
const RideItem = ({ item, onPress }) => (
  <TouchableOpacity style={styles.rideItem} onPress={() => onPress(item.id)}>
    <Text style={styles.rideLocation}>{item.start_location} ➔ {item.end_location}</Text>
    {/* Format date and time nicely */}
    <Text style={styles.rideDateTime}>
      {/* Ensure departure_date is treated as a Date object or parse it */}
      {item.departure_date ? new Date(item.departure_date).toLocaleDateString() : 'N/A'} at {item.departure_time || 'N/A'}
    </Text>
    {/* TODO: Calculate available seats based on total_seats and participants count */}
    <Text style={styles.rideSeats}>{item.total_seats} total seat(s)</Text>
    {/* Add creator info if available later */}
  </TouchableOpacity>
);

// Screen component to display the list of available rides
// This screen now handles both displaying all available rides AND search results
const RidesListScreen = () => {
  const route = useRoute(); // Get route params
  const searchParams = route.params?.searchParams; // Get searchParams if passed
  const navigation = useNavigation();
  const { token } = useAuth(); // Check if user is logged in
  const [rides, setRides] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);
  const [isRefreshing, setIsRefreshing] = useState(false);

  // Function to fetch rides (either all available or based on search params)
  const fetchRides = useCallback(async (params = searchParams) => {
    const isSearch = params && Object.keys(params).length > 0;
    console.log(isSearch ? `Searching rides with params: ${JSON.stringify(params)}` : 'Fetching all available rides...');

    if (!isRefreshing) setIsLoading(true);
    setError(null);
    try {
      let response;
      if (isSearch) {
        // Call the actual searchRides function from the service
        response = await rideService.searchRides(params);
         if (response.status === 'success' && Array.isArray(response.data)) {
            setRides(response.data);
         } else {
            throw new Error(response.message || 'Failed to search rides: Invalid response format');
         }
      } else {
        response = await rideService.listAvailableRides();
         if (response.status === 'success' && Array.isArray(response.data)) {
            setRides(response.data);
         } else {
            throw new Error(response.message || 'Failed to fetch rides: Invalid response format');
         }
      }

    } catch (err) {
      console.error('Error fetching/searching rides:', err);
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
      fetchRides(searchParams); // Pass current searchParams
    }, [fetchRides, searchParams])
  );

  // Handle pull-to-refresh
  const onRefresh = useCallback(async () => {
    console.log('Refreshing rides list...');
    setIsRefreshing(true); // Set refreshing state
    // fetchRides will handle the API call and setting isRefreshing back to false
    await fetchRides(searchParams); // Re-fetch with current params on refresh
  }, [fetchRides]);

  // Handle press on a ride item
  const handleRidePress = (rideId) => {
    if (!token) {
      // If not logged in, prompt to login/signup
      // Consider navigating to login or showing a modal
      alert('Please log in or sign up to view ride details or join a ride.');
      navigation.navigate('Login'); // Navigate to login screen
      return;
    }
    // Navigate to RideDetailScreen if logged in
    navigation.navigate('RideDetail', { rideId });
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
       {/* Add Create Ride button if logged in */}
       {token && (
            <TouchableOpacity
                style={styles.createButton}
                onPress={() => navigation.navigate('CreateRide')}
            >
                <Text style={styles.createButtonText}>+ Create New Ride</Text>
            </TouchableOpacity>
         )}
      <FlatList
        data={rides}
        keyExtractor={(item) => item.id.toString()}
        renderItem={({ item }) => (
          <RideItem item={item} onPress={handleRidePress} />
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
  rideItem: {
    backgroundColor: '#ffffff',
    padding: 15,
    marginBottom: 10,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#eee',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.1,
    shadowRadius: 2,
    elevation: 2,
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
   createButton: {
    backgroundColor: '#28a745', // Green color
    padding: 15,
    borderRadius: 8,
    marginHorizontal: 15,
    marginTop: 15, // Add margin top
    marginBottom: 5, // Add margin bottom
    alignItems: 'center',
  },
  createButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: 'bold',
  },
});

export default RidesListScreen;