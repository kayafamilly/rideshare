// frontend/screens/SearchRideScreen.js
import React, { useState } from 'react';
import { View, Text, StyleSheet, Platform, TouchableOpacity } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import DateTimePicker from '@react-native-community/datetimepicker';

import TextInput from '../components/TextInput'; // Reusable component
import Button from '../components/Button'; // Reusable component

const SearchRideScreen = () => {
  const navigation = useNavigation();
  const [startLocation, setStartLocation] = useState('');
  const [endLocation, setEndLocation] = useState('');
  const [dateTouched, setDateTouched] = useState(false); // Track if date picker was used
  const [date, setDate] = useState(new Date()); // Default to today
  const [showDatePicker, setShowDatePicker] = useState(false);

  const onDateChange = (event, selectedDate) => {
    const currentDate = selectedDate || date;
    setShowDatePicker(Platform.OS === 'ios'); // Keep open on iOS until dismissed
    setDate(currentDate);
    setDateTouched(true); // Mark date as touched
  };

  const showDatepicker = () => {
    setShowDatePicker(true);
  };

  // Format date to YYYY-MM-DD for API query
  const formatDate = (d) => {
    let month = '' + (d.getMonth() + 1);
    let day = '' + d.getDate();
    let year = d.getFullYear();

    if (month.length < 2)
        month = '0' + month;
    if (day.length < 2)
        day = '0' + day;

    return [year, month, day].join('-');
  }

  const handleSearch = () => {
    const searchParams = {};
    const startTrimmed = startLocation.trim();
    const endTrimmed = endLocation.trim();

    if (startTrimmed) {
      searchParams.start_location = startLocation.trim();
    }
    if (endTrimmed) {
      searchParams.end_location = endLocation.trim();
    }
    // Only add date if the user explicitly selected one
    if (dateTouched) {
      searchParams.departure_date = formatDate(date);
    }

    console.log("Navigating to Available Rides with params:", searchParams);
    // Navigate to AvailableRides screen, passing search params
    navigation.navigate('AvailableRides', { searchParams });
  };

  return (
    <View style={styles.container}>
      <Text style={styles.title}>Search for a Ride</Text>

      <TextInput
        label="Starting Location (Optional)"
        value={startLocation}
        onChangeText={setStartLocation}
        placeholder="e.g., Da Nang"
      />
      <TextInput
        label="Ending Location (Optional)"
        value={endLocation}
        onChangeText={setEndLocation}
        placeholder="e.g., Hoi An"
      />

      {/* Date Picker */}
      <View style={styles.datePickerContainer}>
          <Text style={styles.label}>Date (Optional)</Text>
          {/* Button to show picker on Android/iOS */}
          <TouchableOpacity onPress={showDatepicker} style={styles.dateDisplay} testID="searchDateDisplay">
             <Text style={styles.dateText}>{date.toLocaleDateString()}</Text>
          </TouchableOpacity>
          {showDatePicker && (
            <DateTimePicker
              testID="dateTimePicker"
              value={date}
              mode="date"
              is24Hour={true}
              display="default" // Or 'spinner'
              onChange={onDateChange}
              minimumDate={new Date()} // Prevent selecting past dates
            /> // Removed duplicate closing parenthesis
          )}
      </View>


      <Button
        title="Search Rides"
        onPress={handleSearch}
        style={styles.searchButton}
      />
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: 20,
    backgroundColor: '#f8f9fa', // Light background
  },
  title: {
    fontSize: 22,
    fontWeight: 'bold',
    marginBottom: 20,
    textAlign: 'center',
    color: '#333',
  },
  datePickerContainer: {
    width: '100%',
    marginBottom: 20,
  },
  label: {
    marginBottom: 5,
    fontSize: 14,
    color: '#333',
    fontWeight: '500',
  },
  dateDisplay: {
      borderWidth: 1,
      borderColor: '#ccc',
      borderRadius: 8,
      paddingHorizontal: 15,
      paddingVertical: 15, // Increased padding
      backgroundColor: '#fff',
      justifyContent: 'center',
  },
  dateText: {
      fontSize: 16,
      color: '#333',
  },
  searchButton: {
    marginTop: 20,
  }
});

export default SearchRideScreen;