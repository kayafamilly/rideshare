// frontend/screens/SignUpScreen.js
import React, { useState, useCallback } from 'react'; // Import useCallback
import { View, StyleSheet, Text, ScrollView, Alert, KeyboardAvoidingView, Platform, TouchableOpacity } from 'react-native';
import { useNavigation } from '@react-navigation/native'; // To navigate after signup
import DropDownPicker from 'react-native-dropdown-picker'; // Import dropdown
import DateTimePicker from '@react-native-community/datetimepicker'; // Import DateTimePicker

// Import data for pickers
import nationalities from '../config/nationalities.json';
import countryCodes from '../config/countryCodes.json';

import TextInput from '../components/TextInput'; // Reusable component
import Button from '../components/Button'; // Reusable component
import { useAuth } from '../contexts/AuthContext'; // Auth context hook

// Screen component for user registration
const SignUpScreen = () => {
  const navigation = useNavigation();
  const { signup, isLoading } = useAuth(); // Get signup function and loading state

  // State for form fields
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  // Birth Date State
  const [birthDate, setBirthDate] = useState(new Date(2000, 0, 1)); // Default to a reasonable date
  const [showBirthDatePicker, setShowBirthDatePicker] = useState(false);
  // Nationality State
  const [nationalityOpen, setNationalityOpen] = useState(false);
  const [nationalityValue, setNationalityValue] = useState(null);
  const [nationalityItems, setNationalityItems] = useState(nationalities);
  // WhatsApp State
  const [countryCodeOpen, setCountryCodeOpen] = useState(false);
  const [countryCodeValue, setCountryCodeValue] = useState(null); // Store selected prefix e.g., "+84"
  const [countryCodeItems, setCountryCodeItems] = useState(countryCodes);
  const [localWhatsappNumber, setLocalWhatsappNumber] = useState(''); // Store number without prefix

  // State for form errors
  const [errors, setErrors] = useState({});

  // --- Date Picker Logic ---
   const onBirthDateChange = (event, selectedDate) => {
    const currentDate = selectedDate || birthDate;
    setShowBirthDatePicker(Platform.OS === 'ios');
    setBirthDate(currentDate);
    if (errors.birthDate) setErrors(prev => ({...prev, birthDate: null}));
  };

  // --- Dropdown Picker Logic ---
  // Close other pickers when one opens
  const onNationalityOpen = useCallback(() => {
    setCountryCodeOpen(false);
  }, []);
   const onCountryCodeOpen = useCallback(() => {
    setNationalityOpen(false);
  }, []);

  // Format date to YYYY-MM-DD string
  const formatDate = (d) => {
    let month = '' + (d.getMonth() + 1);
    let day = '' + d.getDate();
    let year = d.getFullYear();
    if (month.length < 2) month = '0' + month;
    if (day.length < 2) day = '0' + day;
    return [year, month, day].join('-');
  }

  // --- Validation Logic ---
  const validateForm = () => {
    const newErrors = {};
    if (!email.includes('@')) newErrors.email = 'Please enter a valid email address.';
    if (password.length < 8) newErrors.password = 'Password must be at least 8 characters long.';
    if (password !== confirmPassword) newErrors.confirmPassword = 'Passwords do not match.';
    if (!firstName.trim()) newErrors.firstName = 'First name is required.';
    if (!lastName.trim()) newErrors.lastName = 'Last name is required.';
    // Check if birthdate is reasonably set (e.g., not default or today) - basic check
    if (birthDate.toDateString() === new Date(2000, 0, 1).toDateString()) newErrors.birthDate = 'Please select your birth date.';
    if (!nationalityValue) newErrors.nationality = 'Nationality is required.';
    if (!countryCodeValue) newErrors.whatsapp = 'Country code is required.';
    if (!localWhatsappNumber.trim()) newErrors.whatsapp = 'WhatsApp number is required.';
    // Combine and validate full number (simple check, backend does E.164)
    const fullWhatsapp = countryCodeValue + localWhatsappNumber.trim();
    if (countryCodeValue && localWhatsappNumber.trim() && !/^\+\d{7,}$/.test(fullWhatsapp)) { // Basic check: starts with + and has at least 7 digits after
         newErrors.whatsapp = 'Invalid WhatsApp number format.';
    }


    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  // Handle signup button press
  const handleSignUp = async () => {
    if (!validateForm()) {
      Alert.alert('Validation Error', 'Please check the form fields.');
      return;
    }

    const userData = {
      email: email,
      password: password,
      first_name: firstName.trim(),
      last_name: lastName.trim(),
      birth_date: formatDate(birthDate), // Format date from state
      nationality: nationalityValue,     // Use value from picker state
      whatsapp: countryCodeValue + localWhatsappNumber.trim(), // Combine prefix and local number
    };
// Log the data being sent
console.log("Data being sent to signup:", JSON.stringify(userData, null, 2));

// Call the signup function from AuthContext
// const success = await signup(userData); // REMOVED DUPLICATE LINE
    const success = await signup(userData);

    if (success) {
      // Navigate to Login screen or show a success message
      // Alert is shown within signup function for now
      navigation.navigate('Login'); // Navigate to Login screen after successful signup
    }
    // Error alert is handled within the signup function in AuthContext
  };

  return (
    <KeyboardAvoidingView
      behavior={Platform.OS === "ios" ? "padding" : "height"}
      style={styles.keyboardAvoidingView}
    >
      <ScrollView contentContainerStyle={styles.container}>
        <Text style={styles.title}>Create Account</Text>
        <Text style={styles.subtitle}>Join the RideShare community!</Text>
        {/* Need to wrap pickers in a View with zIndex for dropdowns to overlap */}

        {/* Form Fields */}
        <TextInput
          label="First Name"
          value={firstName}
          onChangeText={setFirstName}
          placeholder="Enter your first name"
          error={errors.firstName}
          autoCapitalize="words"
        />
        <TextInput
          label="Last Name"
          value={lastName}
          onChangeText={setLastName}
          placeholder="Enter your last name"
          error={errors.lastName}
          autoCapitalize="words"
        />
         <TextInput
          label="Email Address"
          value={email}
          onChangeText={setEmail}
          placeholder="your.email@example.com"
          keyboardType="email-address"
          autoCapitalize="none"
          error={errors.email}
        />
         <TextInput
          label="Password"
          value={password}
          onChangeText={setPassword}
          placeholder="Enter your password (min 8 chars)"
          secureTextEntry
          error={errors.password}
        />
        <TextInput
          label="Confirm Password"
          value={confirmPassword}
          onChangeText={setConfirmPassword}
          placeholder="Confirm your password"
          secureTextEntry
          error={errors.confirmPassword}
        />
        {/* Birth Date Picker */}
        <View style={styles.inputContainer}>
            <Text style={styles.label}>Birth Date</Text>
            <TouchableOpacity onPress={() => setShowBirthDatePicker(true)} style={styles.dateDisplay}>
                <Text style={styles.dateText}>{birthDate.toLocaleDateString()}</Text>
            </TouchableOpacity>
            {showBirthDatePicker && (
                <DateTimePicker
                testID="birthDatePicker"
                value={birthDate}
                mode="date"
                display="default"
                onChange={onBirthDateChange}
                maximumDate={new Date(Date.now() - 18 * 365 * 24 * 60 * 60 * 1000)} // Example: 18 years old minimum
                />
            )}
            {errors.birthDate && <Text style={styles.errorText}>{errors.birthDate}</Text>}
        </View>

        {/* Nationality Picker */}
        <View style={[styles.inputContainer, { zIndex: 3000 }]}>
             <Text style={styles.label}>Nationality</Text>
             <DropDownPicker
                open={nationalityOpen}
                value={nationalityValue}
                items={nationalityItems}
                setOpen={setNationalityOpen}
                setValue={setNationalityValue}
                setItems={setNationalityItems}
                onOpen={onNationalityOpen} // Close other pickers
                searchable={true}
                placeholder="Select your nationality"
                listMode="MODAL" // Or "FLATLIST"
                style={styles.dropdown}
                dropDownContainerStyle={styles.dropdownContainer}
                searchPlaceholder="Search..."
                zIndex={3000}
                zIndexInverse={1000}
             />
              {errors.nationality && <Text style={styles.errorText}>{errors.nationality}</Text>}
        </View>

        {/* WhatsApp Input with Prefix Picker */}
        <View style={[styles.inputContainer, { zIndex: 2000 }]}>
             <Text style={styles.label}>WhatsApp Number</Text>
             <View style={styles.whatsappContainer}>
                 <DropDownPicker
                    open={countryCodeOpen}
                    value={countryCodeValue}
                    items={countryCodeItems}
                    setOpen={setCountryCodeOpen}
                    setValue={setCountryCodeValue}
                    setItems={setCountryCodeItems}
                    onOpen={onCountryCodeOpen} // Close other pickers
                    searchable={true}
                    placeholder="+ Code"
                    listMode="MODAL"
                    style={[styles.dropdown, styles.countryCodePicker]}
                    dropDownContainerStyle={styles.dropdownContainer}
                    searchPlaceholder="Search code..."
                    containerStyle={styles.countryCodeContainer} // Style container for width
                    zIndex={2000}
                    zIndexInverse={2000}
                 />
                 <TextInput
                    style={styles.whatsappInput} // Adjust TextInput style
                    containerStyle={{ flex: 1, marginBottom: 0 }} // Adjust container style
                    value={localWhatsappNumber}
                    onChangeText={setLocalWhatsappNumber}
                    placeholder="Your number"
                    keyboardType="phone-pad"
                 />
             </View>
              {errors.whatsapp && <Text style={styles.errorText}>{errors.whatsapp}</Text>}
        </View>

        {/* Signup Button */}
        <Button
          title="Sign Up"
          onPress={handleSignUp}
          loading={isLoading} // Show loading indicator from context
          style={styles.button}
        />

         {/* Link to Login Screen */}
         <TouchableOpacity onPress={() => navigation.navigate('Login')} style={styles.loginLink}>
            <Text style={styles.loginLinkText}>Already have an account? Log In</Text>
         </TouchableOpacity>

      </ScrollView>
    </KeyboardAvoidingView>
  );
};

const styles = StyleSheet.create({
  // Keep existing styles: keyboardAvoidingView, container, title, subtitle, button, loginLink, loginLinkText
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
  container: {
    flexGrow: 1, // Allows content to grow and enable scrolling
  },
  inputContainer: { // Container for label + input/picker
      width: '100%',
      marginBottom: 15,
      // zIndex is set inline for dropdowns
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
      paddingVertical: 15,
      backgroundColor: '#fff',
      justifyContent: 'center',
      minHeight: 50, // Match TextInput height
  },
  dateText: {
      fontSize: 16,
      color: '#333',
  },
  dropdown: {
      borderColor: '#ccc',
      minHeight: 50, // Match TextInput height
  },
  dropdownContainer: {
      borderColor: '#ccc',
  },
  whatsappContainer: {
      flexDirection: 'row',
      alignItems: 'flex-start', // Align items at the top for dropdown
  },
  countryCodeContainer: {
      width: 130, // Fixed width for country code picker
      marginRight: 8,
  },
   countryCodePicker: {
      // Specific styles if needed, inherits from dropdown
      minHeight: 50,
   },
  whatsappInput: {
      flex: 1, // Take remaining space
      // Inherits styles from TextInput component, no need to redefine border etc.
  },
  errorText: {
      marginTop: 4,
      color: 'red',
      fontSize: 12,
  },
  title: { // Keep existing title style
    fontSize: 28,
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
  button: {
    marginTop: 20, // Add some space above the button
  },
   loginLink: {
    marginTop: 20,
  },
  loginLinkText: {
    color: '#007bff', // Blue link color
    fontSize: 14,
  },
});

export default SignUpScreen;