// frontend/screens/CreateRideScreen.js
import React, { useState } from 'react';
import { View, StyleSheet, Text, ScrollView, Alert, KeyboardAvoidingView, Platform, TouchableOpacity } from 'react-native'; // Added TouchableOpacity
import { useNavigation } from '@react-navigation/native';
import DateTimePicker from '@react-native-community/datetimepicker'; // Import DateTimePicker
import RNPickerSelect from 'react-native-picker-select'; // Import Picker

import TextInput from '../components/TextInput'; // Reusable component
import Button from '../components/Button'; // Reusable component
import { rideService } from '../services/api'; // Import ride service
import { useAuth } from '../contexts/AuthContext'; // To ensure user is logged in

// Screen component for creating a new ride
const CreateRideScreen = () => {
  const navigation = useNavigation();
  const { token } = useAuth(); // Get token to ensure user is authenticated
  const [isLoading, setIsLoading] = useState(false);

  // State for form fields
  const [startLocation, setStartLocation] = useState('');
  const [endLocation, setEndLocation] = useState('');
  const [date, setDate] = useState(new Date()); // Use Date object for pickers
  const [time, setTime] = useState(new Date()); // Use Date object for time picker
  const [showDatePicker, setShowDatePicker] = useState(false);
  const [showTimePicker, setShowTimePicker] = useState(false);
  const [totalSeats, setTotalSeats] = useState(null); // Use null or a default number

  // State for form errors
  const [errors, setErrors] = useState({});

  // --- Date/Time Picker Logic ---
  const onDateChange = (event, selectedDate) => {
    const currentDate = selectedDate || date;
    setShowDatePicker(Platform.OS === 'ios');
    setDate(currentDate);
    // Clear date error if any
    if (errors.departureDate) setErrors(prev => ({...prev, departureDate: null}));
  };

  const onTimeChange = (event, selectedTime) => {
    const currentTime = selectedTime || time;
    setShowTimePicker(Platform.OS === 'ios');
    setTime(currentTime);
     // Clear time error if any
    if (errors.departureTime) setErrors(prev => ({...prev, departureTime: null}));
  };

  const showMode = (currentMode) => {
    if (currentMode === 'date') setShowDatePicker(true);
    if (currentMode === 'time') setShowTimePicker(true);
  };

  // Format date/time for display and API
  const formatDate = (d) => {
    let month = '' + (d.getMonth() + 1);
    let day = '' + d.getDate();
    let year = d.getFullYear();
    if (month.length < 2) month = '0' + month;
    if (day.length < 2) day = '0' + day;
    return [year, month, day].join('-');
  }
  const formatTime = (t) => {
      let hours = '' + t.getHours();
      let minutes = '' + t.getMinutes();
      if (hours.length < 2) hours = '0' + hours;
      if (minutes.length < 2) minutes = '0' + minutes;
      return [hours, minutes].join(':');
  }

  // --- Validation Logic ---
  const validateForm = () => {
    const newErrors = {};
    if (!startLocation.trim()) newErrors.startLocation = 'Starting location is required.';
    if (!endLocation.trim()) newErrors.endLocation = 'Ending location is required.';
    if (!totalSeats) newErrors.totalSeats = 'Please select the number of seats.';

    // Combine selected date and time
    const combinedDateTime = new Date(
        date.getFullYear(),
        date.getMonth(),
        date.getDate(),
        time.getHours(),
        time.getMinutes()
    );

    if (combinedDateTime <= new Date()) {
        newErrors.departureTime = 'Departure must be in the future.'; // Show error on time field
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  // Handle create ride button press
  const handleCreateRide = async () => {
    if (!token) {
        Alert.alert('Authentication Error', 'You must be logged in to create a ride.');
        navigation.navigate('Login'); // Redirect to login
        return;
    }

    if (!validateForm()) {
      Alert.alert('Validation Error', 'Please check the form fields.');
      return;
    }

    const rideData = {
      start_location: startLocation.trim(),
      end_location: endLocation.trim(),
      departure_date: formatDate(date),     // Format date from state
      departure_time: formatTime(time),     // Format time from state
      total_seats: totalSeats,              // Use selected number
    };

    setIsLoading(true);
    try {
      const response = await rideService.createRide(rideData);
      console.log('Create ride response:', response); // Expect { status: 'success', data: ride }
      if (response.status === 'success' && response.data) {
        Alert.alert('Success', 'Your ride has been created successfully!');
        // Navigate back to the list or to the detail screen of the new ride
        navigation.goBack(); // Go back to the previous screen (likely RidesList)
        // Or navigate('RideDetail', { rideId: response.data.id });
      } else {
        throw new Error(response.message || 'Failed to create ride: Invalid response');
      }
    } catch (error) {
      console.error('Error creating ride:', error);
      const errorMessage = error.message || 'An unexpected error occurred while creating the ride.';
      Alert.alert('Creation Failed', errorMessage);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : "height"}
      style={styles.keyboardAvoidingView}
    >
      <ScrollView contentContainerStyle={styles.container}>
        <Text style={styles.title}>Create a New Ride</Text>
        <Text style={styles.subtitle}>Share your journey and save costs!</Text>

        {/* Form Fields */}
        <TextInput
          label="Starting Location"
          value={startLocation}
          onChangeText={setStartLocation}
          placeholder="e.g., Da Nang Airport"
          error={errors.startLocation}
        />
        <TextInput
          label="Ending Location"
          value={endLocation}
          onChangeText={setEndLocation}
          placeholder="e.g., Hoi An Ancient Town"
          error={errors.endLocation}
        />
        {/* Date Picker */}
        <View style={styles.pickerContainer}>
            <Text style={styles.label}>Departure Date</Text>
            <TouchableOpacity onPress={() => showMode('date')} style={styles.pickerDisplay}>
                <Text style={styles.pickerText}>{date.toLocaleDateString()}</Text>
            </TouchableOpacity>
            {showDatePicker && (
                <DateTimePicker
                testID="datePicker"
                value={date}
                mode="date"
                display="default"
                onChange={onDateChange}
                minimumDate={new Date()}
                />
            )}
             {errors.departureDate && <Text style={styles.errorText}>{errors.departureDate}</Text>}
        </View>

        {/* Time Picker */}
         <View style={styles.pickerContainer}>
            <Text style={styles.label}>Departure Time</Text>
             <TouchableOpacity onPress={() => showMode('time')} style={styles.pickerDisplay}>
                <Text style={styles.pickerText}>{formatTime(time)}</Text>
            </TouchableOpacity>
            {showTimePicker && (
                <DateTimePicker
                testID="timePicker"
                value={time}
                mode="time"
                is24Hour={true}
                display="default"
                onChange={onTimeChange}
                // minimumDate might not work correctly for time on all platforms
                />
            )}
             {errors.departureTime && <Text style={styles.errorText}>{errors.departureTime}</Text>}
        </View>

        {/* Seats Picker */}
         <View style={styles.pickerContainer}>
             <Text style={styles.label}>Total Seats Offered (1-5)</Text>
             <RNPickerSelect
                onValueChange={(value) => setTotalSeats(value)}
                items={[
                    { label: '1 Seat', value: 1 },
                    { label: '2 Seats', value: 2 },
                    { label: '3 Seats', value: 3 },
                    { label: '4 Seats', value: 4 },
                    { label: '5 Seats', value: 5 },
                ]}
                style={pickerSelectStyles}
                placeholder={{ label: "Select number of seats...", value: null }}
                value={totalSeats}
             />
              {errors.totalSeats && <Text style={styles.errorText}>{errors.totalSeats}</Text>}
         </View>

        {/* Create Ride Button */}
        <Button
          title="Create Ride"
          onPress={handleCreateRide}
          loading={isLoading}
          style={styles.button}
        />
      </ScrollView>
    </KeyboardAvoidingView>
  );
};

const styles = StyleSheet.create({
  keyboardAvoidingView: {
    flex: 1,
  },
  container: {
    flexGrow: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 20,
    backgroundColor: '#f8f9fa',
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    marginBottom: 10,
    color: '#333',
  },
  subtitle: {
    fontSize: 16,
    color: '#666',
    marginBottom: 30,
    textAlign: 'center',
  },
  pickerContainer: {
      width: '100%',
      marginBottom: 15,
  },
  label: { // Reusing label style from SearchRideScreen might be good
      marginBottom: 5,
      fontSize: 14,
      color: '#333',
      fontWeight: '500',
  },
  pickerDisplay: { // Style for the touchable area showing selected value
      borderWidth: 1,
      borderColor: '#ccc',
      borderRadius: 8,
      paddingHorizontal: 15,
      paddingVertical: 15,
      backgroundColor: '#fff',
      justifyContent: 'center',
  },
  pickerText: {
      fontSize: 16,
      color: '#333',
  },
  errorText: { // Error text style
      marginTop: 4,
      color: 'red',
      fontSize: 12,
  },
  button: {
    marginTop: 25,
  },
});

// Styles specifically for RNPickerSelect
const pickerSelectStyles = StyleSheet.create({
  inputIOS: {
    fontSize: 16,
    paddingVertical: 15,
    paddingHorizontal: 15,
    borderWidth: 1,
    borderColor: '#ccc',
    borderRadius: 8,
    color: '#333',
    backgroundColor: '#fff',
    paddingRight: 30, // to ensure the text is never behind the icon
  },
  inputAndroid: {
    fontSize: 16,
    paddingHorizontal: 15,
    paddingVertical: 12,
    borderWidth: 1,
    borderColor: '#ccc',
    borderRadius: 8,
    color: '#333',
    backgroundColor: '#fff',
    paddingRight: 30, // to ensure the text is never behind the icon
  },
  placeholder: {
      color: '#aaa',
  },
  iconContainer: { // Style the dropdown icon position
      top: 15,
      right: 15,
  },
});


export default CreateRideScreen;