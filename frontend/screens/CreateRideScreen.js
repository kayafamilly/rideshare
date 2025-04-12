// frontend/screens/CreateRideScreen.js
import React, { useState, useEffect, useCallback, useRef } from 'react';
import { View, StyleSheet, Text, ScrollView, Alert, KeyboardAvoidingView, Platform, TouchableOpacity, FlatList, ActivityIndicator, Linking } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import DateTimePicker from '@react-native-community/datetimepicker';
import DropDownPicker from 'react-native-dropdown-picker';
import MapView, { Marker, UrlTile, Polyline } from 'react-native-maps';
import debounce from 'lodash.debounce';

import TextInput from '../components/TextInput'; // Assuming this is a custom component
import Button from '../components/Button'; // Assuming this is a custom component
import { rideService } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import { EXPO_PUBLIC_OPENROUTESERVICE_API_KEY } from '@env'; // Import API key
import axios from 'axios'; // For external API calls

// Screen component for creating a new ride
const CreateRideScreen = () => {
  const navigation = useNavigation();
  const { token } = useAuth();
  const [isLoading, setIsLoading] = useState(false); // Loading state for ride creation

  // --- Form State ---
  const [startLocationName, setStartLocationName] = useState('');
  const [endLocationName, setEndLocationName] = useState('');
  const [departureCoords, setDepartureCoords] = useState(null); // { latitude, longitude }
  const [arrivalCoords, setArrivalCoords] = useState(null);     // { latitude, longitude }
  const [date, setDate] = useState(new Date());
  const [time, setTime] = useState(new Date());
  const [showDatePicker, setShowDatePicker] = useState(false);
  const [showTimePicker, setShowTimePicker] = useState(false);
  const [totalSeats, setTotalSeats] = useState(null);
  const [seatsOpen, setSeatsOpen] = useState(false);
  const [seatItems, setSeatItems] = useState([
    { label: '1 Seat', value: 1 },
    { label: '2 Seats', value: 2 },
    { label: '3 Seats', value: 3 },
    { label: '4 Seats', value: 4 },
    { label: '5 Seats', value: 5 },
  ]);
  const [errors, setErrors] = useState({});

  // --- Autocomplete State ---
  const [departureSuggestions, setDepartureSuggestions] = useState([]);
  const [arrivalSuggestions, setArrivalSuggestions] = useState([]);
  const [isDepartureLoading, setIsDepartureLoading] = useState(false);
  const [isArrivalLoading, setIsArrivalLoading] = useState(false);
  const [showDepartureSuggestions, setShowDepartureSuggestions] = useState(false);
  const [showArrivalSuggestions, setShowArrivalSuggestions] = useState(false);

  // --- Map State ---
  const mapRef = useRef(null);
  const [routeCoordinates, setRouteCoordinates] = useState([]);
  const [isRouteLoading, setIsRouteLoading] = useState(false);

  // --- Autocomplete API Call Logic ---
  const fetchSuggestions = async (query, setSuggestions, setIsLoadingSuggestions) => {
    if (!query || query.length < 3) {
      setSuggestions([]);
      return;
    }
    setIsLoadingSuggestions(true);
    try {
      const response = await axios.get(`https://photon.komoot.io/api/?q=${encodeURIComponent(query)}&limit=5&lang=en`);
      const features = response.data.features.filter(f => f.properties?.country && (f.properties?.city || f.properties?.state || f.properties?.county));
      console.log(`Photon suggestions for "${query}":`, features.length);
      setSuggestions(features.map(feature => ({
        ...feature.properties,
        geometry: feature.geometry,
      })));
    } catch (error) {
      console.error(`Error fetching suggestions for "${query}":`, error);
      setSuggestions([]);
    } finally {
      setIsLoadingSuggestions(false);
    }
  };

  const debouncedFetchDepartureSuggestions = useCallback(
    debounce((query) => fetchSuggestions(query, setDepartureSuggestions, setIsDepartureLoading), 300),
    []
  );

  const debouncedFetchArrivalSuggestions = useCallback(
    debounce((query) => fetchSuggestions(query, setArrivalSuggestions, setIsArrivalLoading), 300),
    []
  );

  // --- Map and Routing Logic ---
  const fetchRoute = async (startCoords, endCoords) => {
    if (!startCoords || !endCoords) return;
    console.log("Fetching route from OpenRouteService...");
    setIsRouteLoading(true);
    setRouteCoordinates([]);

    const apiKey = EXPO_PUBLIC_OPENROUTESERVICE_API_KEY;
    if (!apiKey || apiKey === 'YOUR_OPENROUTESERVICE_API_KEY_HERE') {
      console.error("OpenRouteService API Key is missing or placeholder.");
      Alert.alert("Configuration Error", "Map routing service is not configured.");
      setIsRouteLoading(false);
      return;
    }

    const url = `https://api.openrouteservice.org/v2/directions/driving-car`;
    const body = {
      coordinates: [
        [startCoords.longitude, startCoords.latitude],
        [endCoords.longitude, endCoords.latitude]
      ]
    };

    try {
      const response = await axios.post(url, body, {
        headers: { 'Authorization': apiKey, 'Content-Type': 'application/json' }
      });

      if (response.data?.features?.[0]?.geometry?.coordinates) {
        const coordinates = response.data.features[0].geometry.coordinates;
        const formattedCoords = coordinates.map(coord => ({ longitude: coord[0], latitude: coord[1] }));
        console.log(`Route fetched with ${formattedCoords.length} points.`);
        setRouteCoordinates(formattedCoords);

        if (mapRef.current && formattedCoords.length > 0) {
          setTimeout(() => {
            mapRef.current?.fitToCoordinates([startCoords, endCoords, ...formattedCoords], {
              edgePadding: { top: 50, right: 50, bottom: 50, left: 50 }, animated: true,
            });
          }, 100);
        } else if (mapRef.current) {
          setTimeout(() => {
            mapRef.current?.fitToCoordinates([startCoords, endCoords], {
              edgePadding: { top: 50, right: 50, bottom: 50, left: 50 }, animated: true,
            });
          }, 100);
        }
      } else {
        console.warn("No route geometry found in OpenRouteService response:", response.data);
        Alert.alert("Routing Error", "Could not calculate the route.");
      }
    } catch (error) {
      console.error("Error fetching route from OpenRouteService:", error.response?.data || error.message);
      Alert.alert("Routing Error", "Failed to get directions. Please check locations.");
    } finally {
      setIsRouteLoading(false);
    }
  };

  // Effect to fetch route when both coordinates are set
  useEffect(() => {
    if (departureCoords && arrivalCoords) {
      fetchRoute(departureCoords, arrivalCoords);
    } else {
      setRouteCoordinates([]);
    }
  }, [departureCoords, arrivalCoords]);

  // Effect to animate map when only one coordinate is set or both are cleared
  useEffect(() => {
    if (mapRef.current) {
      const coordsToAnimate = departureCoords || arrivalCoords;
      const shouldAnimate = (departureCoords && !arrivalCoords) || (!departureCoords && arrivalCoords);

      if (shouldAnimate && coordsToAnimate) {
        mapRef.current.animateToRegion({
          ...coordsToAnimate, latitudeDelta: 0.05, longitudeDelta: 0.05,
        }, 500);
      }
      // If both are set, the other useEffect handles fitting via fitToCoordinates
      // If both are null, map stays at initialRegion
    }
  }, [departureCoords, arrivalCoords]);

  // --- Date/Time Picker Logic ---
  const onDateChange = (event, selectedDate) => {
    const currentDate = selectedDate || date;
    setShowDatePicker(Platform.OS === 'ios');
    setDate(currentDate);
    if (errors.departureDate) setErrors(prev => ({ ...prev, departureDate: null }));
  };

  const onTimeChange = (event, selectedTime) => {
    const currentTime = selectedTime || time;
    setShowTimePicker(Platform.OS === 'ios');
    setTime(currentTime);
    if (errors.departureTime) setErrors(prev => ({ ...prev, departureTime: null }));
  };

  const showMode = (currentMode) => {
    if (currentMode === 'date') setShowDatePicker(true);
    if (currentMode === 'time') setShowTimePicker(true);
  };

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
    if (!departureCoords) newErrors.startLocation = 'Starting location is required (select from suggestions).';
    if (!arrivalCoords) newErrors.endLocation = 'Ending location is required (select from suggestions).';
    if (!totalSeats) newErrors.totalSeats = 'Please select the number of seats.';

    const combinedDateTime = new Date(
      date.getFullYear(), date.getMonth(), date.getDate(),
      time.getHours(), time.getMinutes()
    );
    if (combinedDateTime <= new Date()) {
      newErrors.departureTime = 'Departure must be in the future.';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  // --- Handle Ride Creation ---
  const handleCreateRide = async () => {
    if (!token) {
      Alert.alert('Authentication Error', 'You must be logged in to create a ride.');
      navigation.navigate('Login');
      return;
    }
    if (!validateForm()) {
      Alert.alert('Validation Error', 'Please check the form fields.');
      return;
    }

    const rideData = {
      departure_location_name: startLocationName.trim(),
      departure_coords: departureCoords ? { longitude: departureCoords.longitude, latitude: departureCoords.latitude } : null,
      arrival_location_name: endLocationName.trim(),
      arrival_coords: arrivalCoords ? { longitude: arrivalCoords.longitude, latitude: arrivalCoords.latitude } : null,
      departure_date: formatDate(date),
      departure_time: formatTime(time),
      total_seats: totalSeats,
    };

    setIsLoading(true);
    try {
      const response = await rideService.createRide(rideData);
      if (response.status === 'success' && response.data) {
        Alert.alert('Success', 'Your ride has been created successfully!');
        navigation.goBack();
      } else {
        throw new Error(response.message || 'Failed to create ride: Invalid response');
      }
    } catch (error) {
      console.error('Error creating ride:', error);
      Alert.alert('Creation Failed', error.message || 'An unexpected error occurred.');
    } finally {
      setIsLoading(false);
    }
  };

  // --- Render Function ---
  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : "height"}
      style={styles.keyboardAvoidingView}
    >
      <ScrollView contentContainerStyle={styles.container} keyboardShouldPersistTaps="handled">
        <Text style={styles.title}>Create a New Ride</Text>
        <Text style={styles.subtitle}>Share your journey and save costs!</Text>

        {/* Location Inputs */}
        <View style={styles.locationInputContainer}>
          <TextInput
            label="Starting Location"
            value={startLocationName}
            onChangeText={(text) => {
              setStartLocationName(text);
              setDepartureCoords(null);
              setRouteCoordinates([]);
              if (text.length > 2) {
                setShowDepartureSuggestions(true);
                debouncedFetchDepartureSuggestions(text);
              } else {
                setShowDepartureSuggestions(false);
                setDepartureSuggestions([]);
                debouncedFetchDepartureSuggestions.cancel();
              }
            }}
            placeholder="Start typing..."
            error={errors.startLocation}
            onBlur={() => setTimeout(() => setShowDepartureSuggestions(false), 150)}
          />
          {isDepartureLoading && <ActivityIndicator style={styles.loadingIndicator} />}
          {showDepartureSuggestions && departureSuggestions.length > 0 && (
            <FlatList
              data={departureSuggestions}
              keyExtractor={(item) => item.osm_id?.toString() ?? Math.random().toString()}
              renderItem={({ item }) => {
                const displayName = `${item.name}, ${item.city || item.state || item.county || ''}, ${item.country}`.replace(/, ,/g, ',').replace(/^, |, $/g, '');
                return (
                  <TouchableOpacity
                    style={styles.suggestionItem}
                    onPress={() => {
                      setStartLocationName(displayName);
                      setDepartureCoords({ latitude: item.geometry.coordinates[1], longitude: item.geometry.coordinates[0] });
                      setShowDepartureSuggestions(false);
                      setDepartureSuggestions([]);
                    }}
                  >
                    <Text>{displayName}</Text>
                  </TouchableOpacity>
                );
              }}
              style={styles.suggestionsList}
            />
          )}
        </View>

        <View style={styles.locationInputContainer}>
          <TextInput
            label="Ending Location"
            value={endLocationName}
            onChangeText={(text) => {
              setEndLocationName(text);
              setArrivalCoords(null);
              setRouteCoordinates([]);
              if (text.length > 2) {
                setShowArrivalSuggestions(true);
                debouncedFetchArrivalSuggestions(text);
              } else {
                setShowArrivalSuggestions(false);
                setArrivalSuggestions([]);
                debouncedFetchArrivalSuggestions.cancel();
              }
            }}
            placeholder="Start typing..."
            error={errors.endLocation}
            onBlur={() => setTimeout(() => setShowArrivalSuggestions(false), 150)}
          />
          {isArrivalLoading && <ActivityIndicator style={styles.loadingIndicator} />}
          {showArrivalSuggestions && arrivalSuggestions.length > 0 && (
            <FlatList
              data={arrivalSuggestions}
              keyExtractor={(item) => item.osm_id?.toString() ?? Math.random().toString()}
              renderItem={({ item }) => {
                 const displayName = `${item.name}, ${item.city || item.state || item.county || ''}, ${item.country}`.replace(/, ,/g, ',').replace(/^, |, $/g, '');
                 return (
                    <TouchableOpacity
                      style={styles.suggestionItem}
                      onPress={() => {
                        setEndLocationName(displayName);
                        setArrivalCoords({ latitude: item.geometry.coordinates[1], longitude: item.geometry.coordinates[0] });
                        setShowArrivalSuggestions(false);
                        setArrivalSuggestions([]);
                      }}
                    >
                      <Text>{displayName}</Text>
                    </TouchableOpacity>
                 );
              }}
              style={styles.suggestionsList}
            />
          )}
        </View>

        {/* Map View */}
        <MapView
          ref={mapRef}
          style={styles.map}
          initialRegion={{ latitude: 16.0479, longitude: 108.2209, latitudeDelta: 10, longitudeDelta: 10 }}
          showsUserLocation={false} // Can be enabled if needed, but requires permission handling here too
        >
          <UrlTile urlTemplate="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" maximumZ={19} flipY={false} />
          {departureCoords && <Marker coordinate={departureCoords} title="Departure" pinColor="green" />}
          {arrivalCoords && <Marker coordinate={arrivalCoords} title="Arrival" pinColor="red" />}
          {routeCoordinates.length > 0 && <Polyline coordinates={routeCoordinates} strokeColor="#007bff" strokeWidth={4} />}
        </MapView>
        <Text style={styles.mapAttribution}>© OpenStreetMap contributors</Text>
        {isRouteLoading && <ActivityIndicator style={styles.mapLoadingIndicator} size="large" />}

        {/* Map Buttons */}
        <View style={styles.mapButtonsContainer}>
          <Button
            title="View Start"
            onPress={() => departureCoords && Linking.openURL(`https://www.openstreetmap.org/?mlat=${departureCoords.latitude}&mlon=${departureCoords.longitude}#map=15/${departureCoords.latitude}/${departureCoords.longitude}`)}
            disabled={!departureCoords} style={styles.mapButton} textStyle={styles.mapButtonText}
          />
          <Button
            title="View Arrival"
            onPress={() => arrivalCoords && Linking.openURL(`https://www.openstreetmap.org/?mlat=${arrivalCoords.latitude}&mlon=${arrivalCoords.longitude}#map=15/${arrivalCoords.latitude}/${arrivalCoords.longitude}`)}
            disabled={!arrivalCoords} style={styles.mapButton} textStyle={styles.mapButtonText}
          />
        </View>

        {/* Date Picker */}
        <View style={styles.pickerContainer}>
          <Text style={styles.label}>Departure Date</Text>
          <TouchableOpacity onPress={() => showMode('date')} style={styles.pickerDisplay} testID="createRideDateDisplay">
            <Text style={styles.pickerText}>{date.toLocaleDateString()}</Text>
          </TouchableOpacity>
          {showDatePicker && (
            <DateTimePicker testID="datePicker" value={date} mode="date" display={Platform.OS === 'ios' ? 'spinner' : 'default'} onChange={onDateChange} minimumDate={new Date()} />
          )}
          {errors.departureDate && <Text style={styles.errorText}>{errors.departureDate}</Text>}
        </View>

        {/* Time Picker */}
        <View style={styles.pickerContainer}>
          <Text style={styles.label}>Departure Time</Text>
          <TouchableOpacity onPress={() => showMode('time')} style={styles.pickerDisplay} testID="createRideTimeDisplay">
            <Text style={styles.pickerText}>{formatTime(time)}</Text>
          </TouchableOpacity>
          {showTimePicker && (
            <DateTimePicker testID="timePicker" value={time} mode="time" is24Hour={true} display={Platform.OS === 'ios' ? 'spinner' : 'default'} onChange={onTimeChange} />
          )}
          {errors.departureTime && <Text style={styles.errorText}>{errors.departureTime}</Text>}
        </View>

        {/* Seats Picker */}
        <View style={[styles.pickerContainer, { zIndex: seatsOpen ? 3000 : 1000 }]}>
          {/* Added zIndex control */}
          <Text style={styles.label}>Total Seats Offered (1-5)</Text>
          <DropDownPicker
            open={seatsOpen}
            value={totalSeats}
            items={seatItems}
            setOpen={setSeatsOpen}
            setValue={(valueCallback) => {
              setTotalSeats(valueCallback);
              if (errors.totalSeats) setErrors(prev => ({ ...prev, totalSeats: null }));
            }}
            setItems={setSeatItems}
            placeholder="Select number of seats..."
            containerStyle={{ height: 50 }} // Fixed height for container
            style={styles.dropdown}
            dropDownContainerStyle={styles.dropdownContainer}
            listMode="SCROLLVIEW"
            zIndex={3000} // Ensure dropdown is above other elements when open
            zIndexInverse={1000} // Ensure container is lower when closed
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
    padding: 20,
    backgroundColor: '#f8f9fa',
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    marginBottom: 10,
    color: '#333',
    textAlign: 'center',
  },
  subtitle: {
    fontSize: 16,
    color: '#666',
    marginBottom: 20,
    textAlign: 'center',
  },
  locationInputContainer: {
    marginBottom: 15,
    position: 'relative',
    zIndex: 2000, // Needs high zIndex for suggestions overlay
  },
  suggestionsList: {
    position: 'absolute',
    top: 80, // Adjust based on TextInput height + label
    left: 0,
    right: 0,
    backgroundColor: 'white',
    borderWidth: 1,
    borderColor: '#ccc',
    borderRadius: 5,
    maxHeight: 150,
    zIndex: 3000, // Highest zIndex for suggestions
  },
  suggestionItem: {
    padding: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#eee',
  },
  loadingIndicator: {
    position: 'absolute',
    right: 10,
    top: 45, // Adjust vertically if needed
  },
  map: {
    width: '100%',
    height: 250,
    marginBottom: 5,
    zIndex: 1000, // Below inputs/suggestions
  },
  mapLoadingIndicator: {
    position: 'absolute',
    top: '50%',
    left: '50%',
    marginTop: -18, // Center indicator on map
    marginLeft: -18,
    zIndex: 1500, // Above map, below suggestions
  },
  mapAttribution: {
    fontSize: 10,
    color: '#666',
    textAlign: 'center',
    marginBottom: 5,
  },
  mapButtonsContainer: {
    flexDirection: 'row',
    justifyContent: 'space-around',
    width: '100%',
    marginBottom: 15,
    zIndex: 1000, // Ensure buttons are clickable
  },
  mapButton: {
    paddingVertical: 8,
    paddingHorizontal: 12,
    flex: 1,
    marginHorizontal: 5,
  },
  mapButtonText: {
    fontSize: 12,
  },
  pickerContainer: {
    width: '100%',
    marginBottom: 15,
    // zIndex handled dynamically for DropDownPicker
  },
  label: {
    marginBottom: 5,
    fontSize: 14,
    color: '#333',
    fontWeight: '500',
  },
  pickerDisplay: {
    borderWidth: 1,
    borderColor: '#ccc',
    borderRadius: 8,
    paddingHorizontal: 15,
    paddingVertical: 15,
    backgroundColor: '#fff',
    justifyContent: 'center',
    minHeight: 50, // Ensure consistent height
  },
  pickerText: {
    fontSize: 16,
    color: '#333',
  },
  errorText: {
    marginTop: 4,
    color: 'red',
    fontSize: 12,
  },
  button: {
    marginTop: 25,
  },
  dropdown: {
    borderColor: '#ccc',
    minHeight: 50,
  },
  dropdownContainer: {
    borderColor: '#ccc',
  },
});

export default CreateRideScreen;