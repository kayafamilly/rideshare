// frontend/screens/SettingsScreen.js
import React, { useState } from 'react'; // Import useState
import { View, Text, StyleSheet, Alert } from 'react-native';
import Button from '../components/Button'; // Reusable component
import { useAuth } from '../contexts/AuthContext'; // To call delete account and get user ID
// No need to import authService here, context handles the API call

const SettingsScreen = () => {
  const { user, deleteAccount } = useAuth(); // Get deleteAccount function from context
  const [isDeleting, setIsDeleting] = useState(false);

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

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Settings</Text>
      {/* TODO: Add Payment Settings section */}
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
});

export default SettingsScreen;