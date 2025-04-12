// frontend/screens/MyRidesScreen.js
import React, { useState, useCallback } from 'react';
import { View, Text, StyleSheet, FlatList, ActivityIndicator, RefreshControl, TouchableOpacity, Alert } from 'react-native';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { rideService } from '../services/api';
import Button from '../components/Button'; // For Delete/Leave buttons

// Simplified RideItem for MyRides screen
const MyRideItem = ({ item, type, onDelete, onLeave, onViewDetails, onViewContacts }) => ( // Added onViewContacts
  <View style={styles.rideItem}>
    <TouchableOpacity onPress={() => onViewDetails(item.id)}>
        <Text style={styles.rideLocation}>{item.start_location} ➔ {item.end_location}</Text>
        <Text style={styles.rideDateTime}>
        {item.departure_date ? new Date(item.departure_date).toLocaleDateString() : 'N/A'} at {item.departure_time ? item.departure_time.substring(0, 5) : 'N/A'}
        </Text>
        {/* Display available seats */}
        <Text style={styles.rideSeats}>
            {(item.total_seats ?? 0) - (item.places_taken ?? 0)} seat(s) available
        </Text>
        <Text style={styles.rideStatus(item.status)}>Status: {item.status}</Text>
    </TouchableOpacity>
    {/* Action Buttons */}
    <View style={styles.actionButtons}>
        {type === 'created' && (
            <Button title="Delete" onPress={() => onDelete(item.id)} style={styles.deleteButton} textStyle={styles.actionButtonText} testID={`deleteButton-${item.id}`} />
        )}
        {type === 'joined' && item.status === 'active' && ( // Actions for active joined rides
            <>
              <Button title="View Contacts" onPress={() => onViewContacts(item.id)} style={styles.contactsButton} textStyle={styles.actionButtonText} />
              <Button title="Leave Ride" onPress={() => onLeave(item.id)} style={styles.leaveButton} textStyle={styles.actionButtonText} testID={`leaveButton-${item.id}`} />
            </>
        )}
         <Button title="Details" onPress={() => onViewDetails(item.id)} style={styles.detailsButton} textStyle={styles.actionButtonText} />
    </View>
  </View>
);


const MyRidesScreen = () => {
    const navigation = useNavigation();
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState(null);
    const [createdRides, setCreatedRides] = useState([]);
    const [joinedRides, setJoinedRides] = useState([]);
    const [isRefreshing, setIsRefreshing] = useState(false);
    const [activeTab, setActiveTab] = useState('created'); // 'created', 'joined', or 'history'

    const [historyRides, setHistoryRides] = useState([]); // State for history

    const fetchData = useCallback(async () => {
        console.log(`Fetching ${activeTab} rides...`);
        if (!isRefreshing) setIsLoading(true);
        setError(null);
        let fetchFunction;
        let setDataFunction;

        switch (activeTab) {
            case 'created':
                fetchFunction = rideService.listCreatedRides;
                setDataFunction = setCreatedRides;
                break;
            case 'joined':
                fetchFunction = rideService.listJoinedRides;
                setDataFunction = setJoinedRides;
                break;
            case 'history':
                fetchFunction = rideService.listHistoryRides; // Use the actual function
                // fetchFunction = async () => { console.warn("History fetch not implemented"); return { status: 'success', data: [] }; }; // Placeholder removed
                setDataFunction = setHistoryRides;
                break;
            default:
                console.error("Invalid tab selected");
                setIsLoading(false);
                setIsRefreshing(false);
                return;
        }

        try {
            const response = await fetchFunction();
            if (response.status === 'success' && Array.isArray(response.data)) {
                setDataFunction(response.data);
            } else {
                throw new Error(response.message || `Failed to fetch ${activeTab} rides`);
            }
        } catch (err) {
            console.error(`Error fetching ${activeTab} rides:`, err);
            setError(err.message || `An error occurred while fetching ${activeTab} rides.`);
            // Clear data on error? Optional.
            // setDataFunction([]);
        } finally {
            setIsLoading(false);
            setIsRefreshing(false);
        }
    }, [activeTab, isRefreshing]);

    useFocusEffect(
        useCallback(() => {
            fetchData();
        }, [fetchData])
    );

    const onRefresh = useCallback(() => {
        setIsRefreshing(true);
        fetchData();
    }, [fetchData]);

    const handleDeleteRide = (rideId) => {
        Alert.alert(
            "Delete Ride",
            "Are you sure you want to delete this ride? If participants have joined and paid, they will NOT be refunded by this action.", // Simplified warning
            [
                { text: "Cancel", style: "cancel" },
                {
                    text: "Delete",
                    style: "destructive",
                    onPress: async () => {
                        console.log(`Attempting to delete ride ${rideId}`);
                        try {
                            const response = await rideService.deleteRide(rideId);
                            if (response.status === 'success') {
                                Alert.alert("Success", response.message);
                                fetchData(); // Refresh list
                            } else {
                                throw new Error(response.message || "Failed to delete ride");
                            }
                        } catch (err) {
                             console.error(`Error deleting ride ${rideId}:`, err);
                             Alert.alert("Error", err.message || "Could not delete ride.");
                        }
                    },
                },
            ]
        );
    };

    const handleLeaveRide = (rideId) => {
         Alert.alert(
            "Leave Ride",
            "Are you sure you want to leave this ride? You will not be refunded.",
            [
                { text: "Cancel", style: "cancel" },
                {
                    text: "Leave",
                    style: "destructive",
                    onPress: async () => {
                         console.log(`Attempting to leave ride ${rideId}`);
                         try {
                            const response = await rideService.leaveRide(rideId);
                             if (response.status === 'success') {
                                Alert.alert("Success", response.message);
                                fetchData(); // Refresh list
                            } else {
                                throw new Error(response.message || "Failed to leave ride");
                            }
                         } catch (err) {
                             console.error(`Error leaving ride ${rideId}:`, err);
                             Alert.alert("Error", err.message || "Could not leave ride.");
                         }
                    },
                },
            ]
        );
    };

     const handleViewDetails = (rideId) => {
        // Navigate to the 'Search' tab first, then to the 'RideDetail' screen within its stack
        navigation.navigate('Search', { screen: 'RideDetail', params: { rideId } });
    };

    const handleViewContacts = async (rideId) => {
        console.log(`Attempting to view contacts for ride ${rideId}`);
        try {
            // Call the API service function
            const response = await rideService.getRideContacts(rideId);
            if (response.status === 'success' && Array.isArray(response.data)) {
                // Format the contact information for display
                const contactsText = response.data.map(contact =>
                    `${contact.first_name || 'N/A'} ${contact.last_name || ''} (${contact.is_creator ? 'Creator' : 'Participant'}): ${contact.whatsapp}`
                ).join('\n');
                Alert.alert("Ride Contacts", contactsText || "No contacts found (this shouldn't happen if authorized).");
            } else {
                throw new Error(response.message || "Failed to fetch contacts");
            }
        } catch (err) {
            console.error(`Error fetching contacts for ride ${rideId}:`, err);
            Alert.alert("Error", err.message || "Could not fetch ride contacts.");
        }
    };


    const renderContent = () => {
        if (isLoading && !isRefreshing) {
            return <ActivityIndicator size="large" color="#007bff" style={styles.loader} />;
        }
        if (error) {
            return <Text style={styles.errorText}>Error: {error}</Text>;
        }

        let data = [];
        let emptyMessage = "";
        switch (activeTab) {
            case 'created':
                data = createdRides;
                emptyMessage = "You haven't created any rides yet.";
                break;
            case 'joined':
                data = joinedRides;
                emptyMessage = "You haven't joined any rides yet.";
                break;
            case 'history':
                data = historyRides;
                emptyMessage = "No past rides found in your history.";
                break;
        }

        return (
            <FlatList
                data={data}
                keyExtractor={(item) => item.id.toString()}
                renderItem={({ item }) => (
                    <MyRideItem
                        item={item}
                        type={activeTab}
                        onDelete={handleDeleteRide}
                        onLeave={handleLeaveRide}
                        onViewDetails={handleViewDetails}
                        onViewContacts={handleViewContacts} // Pass the new handler
                    />
                )}
                style={styles.list}
                ListEmptyComponent={<Text style={styles.emptyText}>{emptyMessage}</Text>}
                refreshControl={
                    <RefreshControl refreshing={isRefreshing} onRefresh={onRefresh} />
                }
            />
        );
    };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>My Rides</Text>
      <View style={styles.tabContainer}>
          <TouchableOpacity
            style={[styles.tabButton, activeTab === 'created' && styles.activeTabButton]}
            onPress={() => setActiveTab('created')}
          >
              <Text style={[styles.tabButtonText, activeTab === 'created' && styles.activeTabButtonText]}>Created By Me</Text>
          </TouchableOpacity>
           <TouchableOpacity
            style={[styles.tabButton, activeTab === 'joined' && styles.activeTabButton]}
            onPress={() => setActiveTab('joined')}
          >
              <Text style={[styles.tabButtonText, activeTab === 'joined' && styles.activeTabButtonText]}>Joined By Me</Text>
          </TouchableOpacity>
      </View>
      <TouchableOpacity
            style={[styles.tabButton, activeTab === 'history' && styles.activeTabButton]}
            onPress={() => setActiveTab('history')}
          >
              <Text style={[styles.tabButtonText, activeTab === 'history' && styles.activeTabButtonText]}>History</Text>
          </TouchableOpacity>
      {renderContent()}
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    // padding: 20, // Padding applied within sections now
    backgroundColor: '#f8f9fa',
  },
  loader: {
      marginTop: 50,
  },
  tabContainer: {
      flexDirection: 'row',
      justifyContent: 'space-around',
      backgroundColor: '#eee',
      paddingVertical: 10,
      marginBottom: 10,
  },
  tabButton: {
      paddingVertical: 8,
      paddingHorizontal: 20,
      borderRadius: 20,
  },
  activeTabButton: {
      backgroundColor: '#007bff',
  },
  tabButtonText: {
      fontSize: 16,
      color: '#333',
  },
  activeTabButtonText: {
      color: '#fff',
      fontWeight: 'bold',
  },
  list: {
      flex: 1,
      paddingHorizontal: 15, // Add horizontal padding for list items
  },
  title: {
    fontSize: 22,
    fontWeight: 'bold',
    marginTop: 10, // Adjust margin
    marginBottom: 15,
    textAlign: 'center',
    color: '#333',
  },
  rideItem: {
    backgroundColor: '#ffffff',
    padding: 15,
    marginBottom: 10,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#eee',
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
    marginBottom: 3,
  },
   rideStatus: (status) => ({
    fontSize: 14,
    fontWeight: 'bold',
    color: status === 'active' ? '#28a745' : (status === 'cancelled' ? '#dc3545' : '#6c757d'),
    textTransform: 'capitalize',
  }),
  actionButtons: {
      flexDirection: 'row',
      justifyContent: 'flex-end', // Align buttons to the right
      marginTop: 10,
      paddingTop: 10,
      borderTopWidth: 1,
      borderTopColor: '#eee',
  },
  actionButtonText: {
      fontSize: 14,
      fontWeight: '500',
  },
  detailsButton: {
      backgroundColor: '#6c757d',
      paddingVertical: 5,
      paddingHorizontal: 12,
      marginLeft: 8,
      minHeight: 0, // Override default minHeight
      width: 'auto', // Override default width
  },
  deleteButton: {
      backgroundColor: '#dc3545',
      paddingVertical: 5,
      paddingHorizontal: 12,
      marginLeft: 8,
      minHeight: 0,
      width: 'auto',
  },
  contactsButton: {
      backgroundColor: '#17a2b8', // Teal color
      paddingVertical: 5,
      paddingHorizontal: 12,
      marginLeft: 8,
      minHeight: 0,
      width: 'auto',
  },
  leaveButton: {
      backgroundColor: '#ffc107',
      paddingVertical: 5,
      paddingHorizontal: 12,
      marginLeft: 8,
      minHeight: 0,
      width: 'auto',
  },
  errorText: {
    color: 'red',
    textAlign: 'center',
    marginTop: 20,
    paddingHorizontal: 20,
  },
  emptyText: {
      textAlign: 'center',
      marginTop: 50,
      fontSize: 16,
      color: '#6c757d',
  }
});

export default MyRidesScreen;