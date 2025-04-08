// frontend/screens/MyRidesScreen.js
import React, { useState, useCallback } from 'react';
import { View, Text, StyleSheet, FlatList, ActivityIndicator, RefreshControl, TouchableOpacity, Alert } from 'react-native';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { rideService } from '../services/api';
import Button from '../components/Button'; // For Delete/Leave buttons

// Simplified RideItem for MyRides screen
const MyRideItem = ({ item, type, onDelete, onLeave, onViewDetails }) => (
  <View style={styles.rideItem}>
    <TouchableOpacity onPress={() => onViewDetails(item.id)}>
        <Text style={styles.rideLocation}>{item.start_location} ➔ {item.end_location}</Text>
        <Text style={styles.rideDateTime}>
        {item.departure_date ? new Date(item.departure_date).toLocaleDateString() : 'N/A'} at {item.departure_time || 'N/A'}
        </Text>
        <Text style={styles.rideSeats}>{item.total_seats} total seat(s)</Text>
        <Text style={styles.rideStatus(item.status)}>Status: {item.status}</Text>
    </TouchableOpacity>
    {/* Action Buttons */}
    <View style={styles.actionButtons}>
        {type === 'created' && (
            <Button title="Delete" onPress={() => onDelete(item.id)} style={styles.deleteButton} textStyle={styles.actionButtonText} />
        )}
        {type === 'joined' && item.status === 'active' && ( // Only allow leaving active rides user joined
            <Button title="Leave Ride" onPress={() => onLeave(item.id)} style={styles.leaveButton} textStyle={styles.actionButtonText} />
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
    const [activeTab, setActiveTab] = useState('created'); // 'created' or 'joined'

    const fetchData = useCallback(async () => {
        console.log(`Fetching ${activeTab} rides...`);
        if (!isRefreshing) setIsLoading(true);
        setError(null);
        try {
            let response;
            if (activeTab === 'created') {
                response = await rideService.listCreatedRides();
                if (response.status === 'success' && Array.isArray(response.data)) {
                    setCreatedRides(response.data);
                } else {
                    throw new Error(response.message || 'Failed to fetch created rides');
                }
            } else { // activeTab === 'joined'
                response = await rideService.listJoinedRides();
                 if (response.status === 'success' && Array.isArray(response.data)) {
                    setJoinedRides(response.data);
                } else {
                    throw new Error(response.message || 'Failed to fetch joined rides');
                }
            }
        } catch (err) {
            console.error(`Error fetching ${activeTab} rides:`, err);
            setError(err.message || `An error occurred while fetching ${activeTab} rides.`);
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
        navigation.navigate('RideDetail', { rideId });
    };


    const renderContent = () => {
        if (isLoading && !isRefreshing) {
            return <ActivityIndicator size="large" color="#007bff" style={styles.loader} />;
        }
        if (error) {
            return <Text style={styles.errorText}>Error: {error}</Text>;
        }

        const data = activeTab === 'created' ? createdRides : joinedRides;
        const emptyMessage = activeTab === 'created'
            ? "You haven't created any rides yet."
            : "You haven't joined any rides yet.";

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