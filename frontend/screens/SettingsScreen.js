// frontend/screens/SettingsScreen.js
import React, { useState } from 'react'; // Import useState
import { View, Text, StyleSheet, Alert, ActivityIndicator } from 'react-native'; // Added ActivityIndicator
import Button from '../components/Button'; // Reusable component
import { useAuth } from '../contexts/AuthContext'; // To call delete account and get user ID
// No need to import authService here, context handles the API call
import { useStripe } from '@stripe/stripe-react-native'; // Import Stripe hook
import { paymentService } from '../services/api'; // Import payment service for SetupIntent

const SettingsScreen = () => {
  const { initPaymentSheet, presentPaymentSheet } = useStripe();
  const { user, deleteAccount, hasPaymentMethod, setHasPaymentMethod } = useAuth(); // Get hasPaymentMethod and setHasPaymentMethod
  const [isDeleting, setIsDeleting] = useState(false);
  const [isSettingUpPayment, setIsSettingUpPayment] = useState(false);
  // Remove local paymentMethodStatus state, rely on context
  // const [paymentMethodStatus, setPaymentMethodStatus] = useState('Checking...');

  const handleDeleteAccount = () => {
    Alert.alert(
      "Delete Account",
      "Are you sure you want to permanently delete your account? This action cannot be undone.",
      [
        { text: "Cancel", style: "cancel" },
        {
          text: "Delete",
          style: "destructive",
          onPress: async () => {
            console.log("Attempting to delete account for user:", user?.id);
            setIsDeleting(true);
            setIsDeleting(true);
            // Call deleteAccount from AuthContext
            const success = await deleteAccount();
            if (success) {
              // Alert and navigation are handled within AuthContext/AppNavigator after logout
              console.log("Account deletion process initiated successfully.");
            } else {
              // Error Alert is handled within AuthContext's deleteAccount
              console.log("Account deletion process failed.");
            }
            // No need for finally here as AuthContext handles state after logout/failure
            // setIsDeleting(false); // This might cause issues if component unmounts due to logout
          },
        },
      ]
    );
  };

  // Function to handle adding/updating payment method
  const handleSetupPaymentMethod = async () => {
    console.log("SettingsScreen: Initiating payment method setup...");
    setIsSettingUpPayment(true);
    try {
      // 1. Create SetupIntent on backend
      console.log("Requesting SetupIntent from backend...");
      const setupIntentResponse = await paymentService.createSetupIntent();
      const clientSecret = setupIntentResponse?.data?.client_secret;
      const customerId = setupIntentResponse?.data?.customer_id;
      console.log(`Received SetupIntent clientSecret: ${clientSecret ? 'OK' : 'MISSING'}, CustomerID: ${customerId}`);

      if (!clientSecret) {
        throw new Error("Failed to get setup client secret from server.");
      }

      // 2. Initialize Payment Sheet
      const { error: initError } = await initPaymentSheet({
        customerId: customerId, // Pass the customer ID received from backend
        // customerEphemeralKeySecret: ephemeralKey, // TODO: Backend needs to provide this if using customerId
        setupIntentClientSecret: clientSecret,
        merchantDisplayName: 'RideShare App', // Your app name
        // allowsDelayedPaymentMethods: true, // Optional
        // returnURL: 'rideshare://stripe-redirect', // Optional for specific flows
      });

      if (initError) {
        console.error('SettingsScreen: Error initializing payment sheet:', initError);
        Alert.alert('Error', `Could not initialize payment setup: ${initError.message}`);
        setIsSettingUpPayment(false);
        return;
      }

      // 3. Present Payment Sheet
      console.log("SettingsScreen: Presenting payment sheet...");
      const { error: presentError } = await presentPaymentSheet();

      if (presentError) {
        console.log("SettingsScreen: Payment sheet presentation failed:", presentError);
        Alert.alert('Error', `Payment setup failed: ${presentError.message}`);
      } else {
        console.log("SettingsScreen: Payment sheet presentation successful.");
        Alert.alert('Success', 'Your payment method has been saved successfully!');
        setHasPaymentMethod(true); // Update context state
        // setPaymentMethodStatus('Saved'); // Remove local state update
      }
    } catch (error) {
      console.error("SettingsScreen: Error in handleSetupPaymentMethod catch block:", error);
      Alert.alert('Error', `An error occurred: ${error.message}`);
    } finally {
      console.log("SettingsScreen: Finished payment method setup attempt.");
      setIsSettingUpPayment(false);
    }
  };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Settings</Text>

      {/* Payment Settings Section */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Payment Method</Text>
        {/* TODO: Display current payment method status/details if available */}
        {/* Display status based on context */}
        <Text style={styles.statusText}>Status: {hasPaymentMethod ? 'Card Saved' : 'No Card Saved'}</Text>
        <Button
          title={hasPaymentMethod ? "Update Payment Method" : "Add Payment Method"} // Corrected syntax
          onPress={handleSetupPaymentMethod}
          loading={isSettingUpPayment}
          disabled={isSettingUpPayment}
        />
      </View>

      {/* TODO: Add Notification Settings section */}

      <View style={styles.section}>
         <Text style={styles.sectionTitle}>Account</Text>
         <Button
            title="Delete My Account"
            onPress={handleDeleteAccount}
            style={styles.deleteButton}
            textStyle={styles.deleteButtonText}
            disabled={isDeleting} // Disable button while deleting
            loading={isDeleting}
         />
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 20,
  },
  title: {
    fontSize: 22,
    fontWeight: 'bold',
    marginBottom: 20,
    textAlign: 'center',
  },
  section: {
      marginTop: 30,
  },
  sectionTitle: {
      fontSize: 18,
      fontWeight: '600',
      marginBottom: 15,
      color: '#333',
      borderBottomWidth: 1,
      borderBottomColor: '#eee',
      paddingBottom: 5,
  },
  deleteButton: {
      backgroundColor: '#dc3545', // Red color
      borderColor: '#dc3545',
  },
  deleteButtonText: {
      color: '#fff',
 },
 statusText: {
   fontSize: 14,
   color: '#555',
   marginBottom: 10,
  },
});

export default SettingsScreen;