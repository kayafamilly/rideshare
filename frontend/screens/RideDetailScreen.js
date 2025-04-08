// frontend/screens/RideDetailScreen.js
import React, { useState, useEffect, useCallback } from 'react';
import { View, Text, StyleSheet, ActivityIndicator, ScrollView, Alert, TouchableOpacity } from 'react-native';
import { useRoute, useNavigation } from '@react-navigation/native';
import { useStripe } from '@stripe/stripe-react-native'; // Import Stripe hook

import Button from '../components/Button'; // Reusable component
import { rideService, paymentService } from '../services/api'; // Import ride and payment services
import { useAuth } from '../contexts/AuthContext'; // To get current user ID

// Screen component to display details of a single ride
const RideDetailScreen = () => {
  const route = useRoute();
  const navigation = useNavigation();
  const { user, token } = useAuth(); // Get current user and token
  const { initPaymentSheet, presentPaymentSheet } = useStripe(); // Stripe hooks
  const { rideId } = route.params; // Get rideId passed via navigation

  const [ride, setRide] = useState(null);
  const [isLoading, setIsLoading] = useState(true); // Loading ride details
  const [isProcessingPayment, setIsProcessingPayment] = useState(false); // Loading state for payment process
  const [error, setError] = useState(null);
  const [contacts, setContacts] = useState([]); // State for contacts
  const [isLoadingContacts, setIsLoadingContacts] = useState(false);
  const [contactsError, setContactsError] = useState(null);

  // Function to fetch ride details
  const fetchRideDetails = useCallback(async () => {
    if (!rideId) {
      setError("Ride ID is missing.");
      setIsLoading(false);
      return;
    }
    console.log(`Fetching details for ride ID: ${rideId}`);
    setIsLoading(true);
    setError(null);
    try {
      // Ensure user is logged in to view details (as per backend route protection)
      if (!token) throw new Error("Authentication required to view ride details.");

      const response = await rideService.getRideDetails(rideId);
      console.log('Ride details fetched:', response.data); // Assuming { status: 'success', data: ride }
      if (response.status === 'success' && response.data) {
        setRide(response.data);
      } else {
        throw new Error(response.message || 'Failed to fetch ride details: Invalid response');
      }
    } catch (err) {
      console.error('Error fetching ride details:', err);
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
  }, [rideId, navigation, token]); // Add token dependency

  // Fetch details when the component mounts or rideId changes
  useEffect(() => {
    fetchRideDetails();
  }, [fetchRideDetails]);

  // Handle "Join & Pay" button press
  const handleJoinAndPay = async () => {
    if (!ride || !user) return;

    setIsProcessingPayment(true);
    setError(null); // Clear previous errors

    try {
      // Step 1: Attempt to join the ride (creates participant record with pending_payment)
      console.log(`Attempting to join ride ${ride.id}`);
      const joinResponse = await rideService.joinRide(ride.id);
      console.log('Join ride response:', joinResponse); // Expect { status: 'success', data: participant, message: '...' }

      // Check if joining was successful before proceeding
      if (joinResponse.status !== 'success' || !joinResponse.data) {
        // Use the message from the join response if available
        throw new Error(joinResponse.message || 'Failed to join ride. Cannot proceed to payment.');
      }
      // Optional: Show a quick confirmation before payment starts
      // Alert.alert('Ride Joined (Pending Payment)', 'Now proceeding to payment...');

      // Step 2: Create the Payment Intent (backend now expects a pending participant)
      console.log(`Creating payment intent for ride ${ride.id}`);
      const intentResponse = await paymentService.createPaymentIntent(ride.id);
      console.log('Payment Intent response:', intentResponse); // Expect { status: 'success', data: { client_secret, ... } }

      if (intentResponse.status !== 'success' || !intentResponse.data?.client_secret) {
        // If PI creation fails after joining, the user is left in 'pending_payment' state.
        // They might need a way to retry payment later.
        throw new Error(intentResponse.message || 'Failed to initialize payment after joining.');
      }

      const clientSecret = intentResponse.data.client_secret;

      // Step 3: Initialize the Payment Sheet
      const { error: initError } = await initPaymentSheet({
        merchantDisplayName: "RideShare Demo", // Your app/merchant name
        paymentIntentClientSecret: clientSecret,
        // allowsDelayedPaymentMethods: true, // Optional
        // defaultBillingDetails: { name: `${user.first_name} ${user.last_name}` }, // Optional prefill
      });

      if (initError) {
        console.error('Error initializing payment sheet:', initError);
        throw new Error(`Payment sheet initialization failed: ${initError.message}`);
      }

      // Step 4: Present the Payment Sheet
      const { error: paymentError } = await presentPaymentSheet();

      if (paymentError) {
        console.error('Error presenting payment sheet:', paymentError);
        if (paymentError.code === 'Canceled') {
          Alert.alert('Payment Canceled', 'Your participation is pending payment. You can try paying again later.');
          // Stay on the screen or navigate back? Stay might be better.
        } else {
          throw new Error(`Payment failed: ${paymentError.message}`);
        }
      } else {
        // Payment successful! Webhook will handle confirmation on backend.
        Alert.alert('Payment Successful!', 'Your payment was successful. Your participation will be confirmed shortly.');
        // Navigate back or update UI to show 'pending confirmation' state?
        navigation.goBack();
      }

    } catch (error) {
      console.error('Error during join/payment process:', error);
      const errorMessage = error.message || 'An unexpected error occurred.';
      Alert.alert('Payment Process Failed', errorMessage);
    } finally {
      setIsProcessingPayment(false);
    }
  };

  // Handle "View Contacts" button press
  const handleViewContacts = async () => {
    if (!ride || !user) return;

    setIsLoadingContacts(true);
    setContactsError(null);
    setContacts([]); // Clear previous contacts

    try {
      console.log(`Fetching contacts for ride ${ride.id}`);
      const response = await rideService.getRideContacts(ride.id);
      console.log('Contacts response:', response); // Expect { status: 'success', data: [...] }

      if (response.status === 'success' && Array.isArray(response.data)) {
        setContacts(response.data);
        if (response.data.length === 0) {
            // This might happen if only the creator exists and hasn't paid yet,
            // or if the requesting user isn't found in the list (shouldn't happen with backend logic)
             setContactsError("No confirmed participant contacts found yet.");
        }
      } else {
        throw new Error(response.message || 'Failed to fetch contacts.');
      }
    } catch (error) {
      console.error('Error fetching contacts:', error);
      const errorMessage = error.message || 'An unexpected error occurred while fetching contacts.';
      setContactsError(errorMessage);
      // Alert might be annoying if user just isn't authorized yet
      // Alert.alert('Could Not Fetch Contacts', errorMessage);
    } finally {
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
  const canJoin = ride.status === 'open' && ride.available_seats > 0 && !isCreator;
  // Show contacts button only if the current user is the creator (simplest check for now)
  const showContactsButton = isCreator;

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
        <Text style={styles.value}>{ride.departure_time || 'N/A'}</Text>
      </View>
      <View style={styles.detailItem}>
        <Text style={styles.label}>Seats Available:</Text>
        <Text style={styles.value}>{ride.available_seats}</Text>
      </View>
      <View style={styles.detailItem}>
        <Text style={styles.label}>Status:</Text>
        <Text style={[styles.value, styles.statusValue(ride.status)]}>{ride.status}</Text>
      </View>
      {/* TODO: Add Creator Info here later */}
      {/* <View style={styles.detailItem}>
        <Text style={styles.label}>Created By:</Text>
        <Text style={styles.value}>{ride.creator?.first_name || 'Unknown'}</Text>
      </View> */}


      {/* Join Button - Conditionally Rendered */}
      {canJoin && (
        <Button
          title="Join & Pay (2 €)" // Updated button title
          onPress={handleJoinAndPay}
          loading={isProcessingPayment} // Use new loading state
          style={styles.joinButton}
        />
      )}
      {/* Info Texts */}
      {isCreator && (
        <Text style={styles.infoText}>This is your ride. You can view confirmed participants' contacts.</Text>
      )}
      {!isCreator && !canJoin && ride.status === 'open' && ride.available_seats <= 0 && (
        <Text style={styles.infoText}>This ride is full.</Text>
      )}
      {!isCreator && !canJoin && ride.status !== 'open' && (
        <Text style={styles.infoText}>This ride is not available for joining (Status: {ride.status}).</Text>
      )}
      {/* TODO: Add check/info if user has already joined */}


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
                {contact.first_name || 'N/A'} {contact.last_name || ''} {contact.is_creator ? '(Creator)' : ''}
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
    color: status === 'open' ? '#28a745' : (status === 'full' || status === 'closed' || status === 'cancelled' ? '#dc3545' : '#6c757d'),
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