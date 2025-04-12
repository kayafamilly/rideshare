// frontend/screens/RideDetailScreen.js
import React, { useState, useEffect, useCallback } from 'react';
import { View, Text, StyleSheet, ActivityIndicator, ScrollView, Alert, TouchableOpacity } from 'react-native';
import { useRoute, useNavigation, useFocusEffect } from '@react-navigation/native'; // Added useFocusEffect
import { useStripe } from '@stripe/stripe-react-native'; // Import Stripe hook

import Button from '../components/Button'; // Reusable component
import { rideService, paymentService } from '../services/api'; // Import ride and payment services
import { useAuth } from '../contexts/AuthContext'; // To get current user ID

// Screen component to display details of a single ride
const RideDetailScreen = () => {
  const route = useRoute();
  const navigation = useNavigation();
  const { user, token, hasPaymentMethod: contextHasPaymentMethod } = useAuth(); // Get payment status from context
  const { initPaymentSheet, presentPaymentSheet } = useStripe(); // Stripe hooks
  const { rideId } = route.params; // Get rideId passed via navigation

  const [ride, setRide] = useState(null);
  const [isLoading, setIsLoading] = useState(true); // Loading ride details
  const [isProcessingPayment, setIsProcessingPayment] = useState(false); // Loading state for payment process
  const [error, setError] = useState(null);
  const [contacts, setContacts] = useState([]); // State for contacts
  const [isLoadingContacts, setIsLoadingContacts] = useState(false);
  const [contactsError, setContactsError] = useState(null);
  const [myStatus, setMyStatus] = useState('loading'); // loading, not_participant, pending_payment, active, left, etc.

  // Function to fetch ride details
  const fetchRideDetails = useCallback(async () => {
    if (!rideId) {
      setError("Ride ID is missing.");
      setIsLoading(false);
      return;
    }
    console.log(`RideDetailScreen: Fetching details for ride ID: ${rideId}`);
    setIsLoading(true);
    setError(null);
    try {
      // Ensure user is logged in to view details (as per backend route protection)
      if (!token) throw new Error("Authentication required to view ride details.");

      const response = await rideService.getRideDetails(rideId);
      console.log('Ride details fetched:', response.data); // Assuming { status: 'success', data: ride }
      if (response.status === 'success' && response.data) {
        console.log("RideDetailScreen: Ride details successfully fetched.");
        setRide(response.data);
      } else {
        throw new Error(response.message || 'Failed to fetch ride details: Invalid response');
      }
    } catch (err) {
      console.error('RideDetailScreen: Error fetching ride details:', err);
      const errorMessage = err.message || 'An error occurred while fetching ride details.';
      setError(errorMessage);
      // If ride not found (404), the API interceptor should reject with { message: 'ride not found' }
      if (errorMessage === 'ride not found') {
         // Optionally navigate back or show specific message
         Alert.alert('Not Found', 'This ride could not be found.');
         navigation.goBack();
      } else if (errorMessage.includes('Unauthorized') || errorMessage.includes('Token')) {
          // Handle case where token is invalid/expired during fetch
          Alert.alert('Session Expired', 'Please log in again to view ride details.');
          // Potentially trigger logout or navigate to login
          navigation.navigate('Login');
      }
    } finally {
      setIsLoading(false);
    }
  }, [rideId, navigation, token]);

  // Fetch details and participation status when the screen focuses or rideId changes
  useFocusEffect(
    useCallback(() => {
      let isActive = true; // Flag to prevent state updates if component unmounts

      const loadData = async () => {
        // Wait for user, token, and rideId before proceeding
        if (!rideId || !token || !user) {
          console.log("RideDetailScreen: Waiting for rideId, token, or user...");
          if (isActive) setMyStatus('loading'); // Keep loading state until all prerequisites are met
          return; // Exit early if prerequisites are not met
        }

        if (isActive) setMyStatus('loading');
        await fetchRideDetails(); // Fetch ride details first

        // Then fetch participation status
        try {
          console.log(`RideDetailScreen: Fetching participation status for user ${user?.id} on ride ${rideId}`);
          const statusResponse = await rideService.getMyParticipationStatus(rideId);
          if (isActive && statusResponse.status === 'success' && statusResponse.data) {
            console.log(`RideDetailScreen: Participation status received: ${statusResponse.data.participation_status}`);
            setMyStatus(statusResponse.data.participation_status);
          } else {
             if (isActive) setMyStatus('error'); // Indicate error fetching status
             console.warn("RideDetailScreen: Failed to get participation status:", statusResponse?.message);
          }
        } catch (err) {
           if (isActive) setMyStatus('error');
           console.error("RideDetailScreen: Error fetching participation status:", err);
        }
      };

      loadData();

      return () => {
        isActive = false; // Cleanup function to set flag false when component unmounts
      };
    }, [fetchRideDetails, rideId, token, user]) // Depend on the whole user object
  );

  // Handle "Join Ride" button press (using automatic payment flow)
  const handleJoinRide = async () => {
    if (!ride || !user) return;

    console.log(`RideDetailScreen: handleJoinRide initiated for ride ${ride.id}`);
    setIsProcessingPayment(true); // Use existing state for loading indicator
    setError(null);

    // Check payment method status from context
    const hasPaymentMethod = contextHasPaymentMethod;
    console.log(`RideDetailScreen: User has payment method? ${hasPaymentMethod}`);

    if (hasPaymentMethod) {
      // Attempt automatic payment
      console.log(`RideDetailScreen: Attempting automatic join API call for ride ${ride.id}`);
      Alert.alert("Processing Payment", "Attempting to join ride using your saved payment method...");
      try {
        const response = await rideService.joinRideAutomatically(ride.id);
        if (response.status === 'success') {
          console.log(`RideDetailScreen: Automatic join API call successful for ride ${ride.id}`);
          Alert.alert("Success", response.message || "Successfully joined ride and payment processed.");
          // Refresh details or navigate back? Refreshing might show updated status/seats.
          fetchRideDetails(); // Re-fetch details
        } else {
          // Should be caught by catch block if backend returns non-2xx
          console.warn(`RideDetailScreen: Automatic join API call failed (status not success) for ride ${ride.id}:`, response.message);
          throw new Error(response.message || "Could not join the ride automatically.");
        }
      } catch (apiError) {
        console.error(`RideDetailScreen: Error calling joinRideAutomatically for ride ${ride.id}:`, apiError);
        Alert.alert("Error", apiError.message || "An error occurred while trying to join the ride.");
      } finally {
        setIsProcessingPayment(false);
      }
    } else {
      console.log(`RideDetailScreen: No payment method found, prompting user to go to Settings.`);
      // Redirect to Settings if no payment method
      Alert.alert(
        "Payment Method Required",
        "Please add a payment method in Settings before joining a ride.",
        [
          { text: "Go to Settings", onPress: () => navigation.navigate('Settings') },
          { text: "Cancel", style: "cancel" }
        ]
      );
      setIsProcessingPayment(false); // Stop loading indicator
      console.log("RideDetailScreen: handleJoinRide finished (no payment method).");
    }
  };

  // Handle "View Contacts" button press
  const handleViewContacts = async () => {
    if (!ride || !user) return;

    console.log(`RideDetailScreen: handleViewContacts initiated for ride ${ride.id}`);
    setIsLoadingContacts(true);
    setContactsError(null);
    setContacts([]); // Clear previous contacts

    try {
      console.log(`RideDetailScreen: Fetching contacts for ride ${ride.id}`);
      const response = await rideService.getRideContacts(ride.id);
      console.log('Contacts response:', response); // Expect { status: 'success', data: [...] }

      if (response.status === 'success' && Array.isArray(response.data)) {
        console.log(`RideDetailScreen: Successfully fetched ${response.data.length} contacts for ride ${ride.id}`);
        setContacts(response.data);
        if (response.data.length === 0) {
            // This might happen if only the creator exists and hasn't paid yet,
            // or if the requesting user isn't found in the list (shouldn't happen with backend logic)
             setContactsError("No confirmed participant contacts found yet.");
        }
      } else {
        console.warn(`RideDetailScreen: Fetch contacts API call failed (status not success) for ride ${ride.id}:`, response.message);
        throw new Error(response.message || 'Failed to fetch contacts.');
      }
    } catch (error) {
      console.error(`RideDetailScreen: Error calling getRideContacts for ride ${ride.id}:`, error);
      const errorMessage = error.message || 'An unexpected error occurred while fetching contacts.';
      setContactsError(errorMessage);
      // Alert might be annoying if user just isn't authorized yet
      // Alert.alert('Could Not Fetch Contacts', errorMessage);
    } finally {
      console.log(`RideDetailScreen: Finished handleViewContacts attempt for ride ${ride.id}`);
      setIsLoadingContacts(false);
    }
  };


  // Render loading indicator
  if (isLoading) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator size="large" color="#007bff" />
        <Text>Loading ride details...</Text>
      </View>
    );
  }

  // Render error message
  if (error) {
    return (
      <View style={styles.centered}>
        <Text style={styles.errorText}>Error: {error}</Text>
         <TouchableOpacity onPress={fetchRideDetails} style={styles.retryButton}>
           <Text style={styles.retryButtonText}>Retry</Text>
        </TouchableOpacity>
      </View>
    );
  }

  // Render ride not found (should be handled by error state now)
  if (!ride) {
    // This case might be redundant if error handling covers not found state
    return (
      <View style={styles.centered}>
        <Text>Ride details could not be loaded.</Text>
      </View>
    );
  }

  // Determine button/info states
  const isCreator = ride.user_id === user?.id;
  // We don't know participant status easily here, so 'canJoin' remains the primary check for the join button.
  const placesTaken = ride.places_taken || 0; // Use places_taken from backend, default to 0
  const availableSeats = ride.total_seats - placesTaken;
  // Determine if user can join based on ride status, seats, creator status, AND participation status
  const canJoin = ride.status === 'active' && availableSeats > 0 && !isCreator && myStatus === 'not_participant';
  // Show contacts button if user is creator OR if they are an active participant (need participant status)
  // For now, let's assume we need to fetch participant status separately or rely on backend authorization in getRideContacts
  const showContactsButton = isCreator || myStatus === 'active'; // Show if creator or active participant

  // Render the ride details
  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>Ride Details</Text>

      <View style={styles.detailItem}>
        <Text style={styles.label}>From:</Text>
        <Text style={styles.value}>{ride.start_location}</Text>
      </View>
      <View style={styles.detailItem}>
        <Text style={styles.label}>To:</Text>
        <Text style={styles.value}>{ride.end_location}</Text>
      </View>
      <View style={styles.detailItem}>
        <Text style={styles.label}>Date:</Text>
        <Text style={styles.value}>{ride.departure_date ? new Date(ride.departure_date).toLocaleDateString() : 'N/A'}</Text>
      </View>
      <View style={styles.detailItem}>
        <Text style={styles.label}>Time:</Text>
        {/* Format time to HH:MM */}
        <Text style={styles.value}>{ride.departure_time ? ride.departure_time.substring(0, 5) : 'N/A'}</Text>
      </View>
      <View style={styles.detailItem}>
        <Text style={styles.label}>Seats Available:</Text>
        <Text style={styles.value}>{availableSeats}</Text>
      </View>
      <View style={styles.detailItem}>
        <Text style={styles.label}>Status:</Text>
        <Text style={[styles.value, styles.statusValue(ride.status)]}>{ride.status}</Text>
      </View>
      {/* Add Creator Info */}
      <View style={styles.detailItem}>
        <Text style={styles.label}>Created By:</Text>
        {/* Display only the first name */}
        <Text style={styles.value}>{ride.creator_first_name || 'Unknown'}</Text>
        {/* Assuming backend already sends only first name as creator_first_name */}
        {/* If backend sends full name, we might need ride.creator_full_name.split(' ')[0] */}
      </View>


      {/* Join Button - Conditionally Rendered */}
      {canJoin && (
        <Button
          title="Join Ride (2 €)" // Simpler title
          onPress={handleJoinRide} // Use the new handler
          loading={isProcessingPayment} // Use new loading state
          style={styles.joinButton}
        />
      )}
      {/* Info Texts */}
      {isCreator && (
        <Text style={styles.infoText}>This is your ride. You can view confirmed participants' contacts.</Text>
      )}
      {!isCreator && !canJoin && ride.status === 'active' && availableSeats <= 0 && (
        <Text style={styles.infoText}>This ride is full.</Text>
      )}
      {!isCreator && !canJoin && ride.status !== 'active' && (
        <Text style={styles.infoText}>This ride is not available for joining (Status: {ride.status}).</Text>
      )}
      {/* Show status if user is a participant */}
      {!isCreator && myStatus !== 'not_participant' && myStatus !== 'loading' && myStatus !== 'error' && (
        <Text style={styles.infoText}>Your Status: <Text style={styles.statusValue(myStatus)}>{myStatus.replace('_', ' ')}</Text></Text>
      )}
      {/* Add specific message if payment is pending */}
      {myStatus === 'pending_payment' && (
         <Text style={styles.infoText}>Your payment is pending. You might need to retry.</Text>
         // TODO: Add a "Retry Payment" button?
      )}


      {/* View Contacts Button - Show only for creator for now */}
      {showContactsButton && (
        <Button
          title={contacts.length > 0 ? "Hide Contacts" : "View Confirmed Participants"}
          onPress={contacts.length > 0 ? () => setContacts([]) : handleViewContacts} // Toggle view/hide
          loading={isLoadingContacts}
          style={styles.viewContactsButton}
        />
      )}

      {/* Display Contacts List */}
      {isLoadingContacts && <ActivityIndicator style={{ marginTop: 15 }} color="#007bff"/>}
      {contactsError && !isLoadingContacts && (
          <Text style={[styles.errorText, { marginTop: 15 }]}>{contactsError}</Text>
      )}
      {contacts.length > 0 && !isLoadingContacts && (
        <View style={styles.contactsContainer}>
          <Text style={styles.contactsTitle}>Confirmed Participants:</Text>
          {contacts.map((contact) => (
            <View key={contact.user_id} style={styles.contactItem}>
              <Text style={styles.contactName}>
                {contact.first_name || 'N/A'} {contact.is_creator ? '(Creator)' : ''}
              </Text>
              <Text style={styles.contactWhatsapp}>WhatsApp: {contact.whatsapp}</Text>
            </View>
          ))}
        </View>
      )}


    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flexGrow: 1,
    padding: 20,
    backgroundColor: '#fff',
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 20,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    marginBottom: 25,
    textAlign: 'center',
    color: '#333',
  },
  detailItem: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 15,
    paddingVertical: 10,
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  label: {
    fontSize: 16,
    fontWeight: '500',
    color: '#555',
    flex: 1, // Take up some space
  },
  value: {
    fontSize: 16,
    color: '#333',
    flex: 2, // Take up more space
    textAlign: 'right',
  },
  statusValue: (status) => ({ // Function to return style based on status
    fontWeight: 'bold',
    // Adjust color based on more statuses if needed
    color: status === 'active' ? '#28a745' : (status === 'cancelled' || status === 'left' ? '#dc3545' : (status === 'pending_payment' ? '#ffc107' : '#6c757d')),
    textTransform: 'capitalize',
  }),
  joinButton: {
    marginTop: 30,
    backgroundColor: '#007bff', // Blue for join action
  },
  viewContactsButton: {
    marginTop: 15,
    backgroundColor: '#17a2b8', // Teal color for view contacts
  },
  infoText: {
      marginTop: 30,
      textAlign: 'center',
      fontSize: 16,
      color: '#6c757d', // Grey info text
  },
  errorText: {
    color: 'red',
    marginBottom: 10,
    textAlign: 'center',
  },
  retryButton: {
      marginTop: 15,
      backgroundColor: '#6c757d', // Grey retry button
      paddingVertical: 10,
      paddingHorizontal: 20,
      borderRadius: 5,
  },
  retryButtonText: {
      color: '#fff',
      fontSize: 16,
  },
  // Styles for contacts list
  contactsContainer: {
    marginTop: 25,
    paddingTop: 15,
    borderTopWidth: 1,
    borderTopColor: '#eee',
  },
  contactsTitle: {
    fontSize: 18,
    fontWeight: 'bold',
    marginBottom: 10,
    color: '#333',
  },
  contactItem: {
    marginBottom: 10,
    padding: 10,
    backgroundColor: '#f8f9fa',
    borderRadius: 5,
  },
  contactName: {
    fontSize: 16,
    fontWeight: '500',
  },
  contactWhatsapp: {
    fontSize: 14,
    color: '#555',
    marginTop: 3,
  },
});

export default RideDetailScreen;