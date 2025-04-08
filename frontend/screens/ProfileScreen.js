// frontend/screens/ProfileScreen.js
import React, { useState, useEffect } from 'react'; // Import useState, useEffect
import { View, Text, StyleSheet, TouchableOpacity, Alert, ScrollView } from 'react-native'; // Added ScrollView
import { useAuth } from '../contexts/AuthContext'; // To get user info
import { authService } from '../services/api'; // Import authService for updateProfile
import Button from '../components/Button'; // Import Button component
import TextInput from '../components/TextInput'; // Import TextInput for editing
// Import pickers for nationality/date
import DropDownPicker from 'react-native-dropdown-picker';
import DateTimePicker from '@react-native-community/datetimepicker';
import nationalities from '../config/nationalities.json';
// Note: Country code picker for WhatsApp is not added here for simplicity, assuming full number editing

const ProfileScreen = () => {
  const { user, logout, updateUserState } = useAuth(); // Use updateUserState from context
  const [isEditing, setIsEditing] = useState(false); // State to toggle edit mode
  const [isSaving, setIsSaving] = useState(false);
  const [errors, setErrors] = useState({});

  // State for editable fields, initialized with user data
  const [firstName, setFirstName] = useState('');
  const [lastName, setLastName] = useState('');
  const [birthDate, setBirthDate] = useState(''); // Store as YYYY-MM-DD string for input
  const [nationality, setNationality] = useState(''); // Stores the string value
  const [whatsapp, setWhatsapp] = useState('');
  // States for pickers
  const [showBirthDatePicker, setShowBirthDatePicker] = useState(false);
  const [nationalityOpen, setNationalityOpen] = useState(false);
  const [nationalityItems, setNationalityItems] = useState(nationalities); // Load items

  // Effect to initialize form state when user data is available/changes
  useEffect(() => {
    if (user) {
      setFirstName(user.first_name || '');
      setLastName(user.last_name || '');
      setBirthDate(user.birth_date ? user.birth_date.split('T')[0] : ''); // Format to YYYY-MM-DD
      setNationality(user.nationality || '');
      setWhatsapp(user.whatsapp || '');
    }
    // Initialize picker value if user has nationality
    setNationalityValue(user.nationality || null);

  }, [user]); // Re-run when user object changes

  // Add state for nationality picker value separate from the text input state
  const [nationalityValue, setNationalityValue] = useState(null);

  // Date Picker handler
   const onBirthDateChange = (event, selectedDate) => {
    const currentDate = selectedDate || new Date(birthDate); // Use current state if no date selected
    setShowBirthDatePicker(Platform.OS === 'ios');
    // Format to YYYY-MM-DD string immediately for the state
    setBirthDate(formatDate(currentDate));
    if (errors.birthDate) setErrors(prev => ({...prev, birthDate: null}));
  };

   // Format date helper (needed for date picker)
   const formatDate = (d) => {
    if (!d) return '';
    let dateObj = (typeof d === 'string') ? new Date(d + 'T00:00:00') : d; // Handle string or Date object
    if (isNaN(dateObj.getTime())) return ''; // Invalid date
    let month = '' + (dateObj.getMonth() + 1);
    let day = '' + dateObj.getDate();
    let year = dateObj.getFullYear();
    if (month.length < 2) month = '0' + month;
    if (day.length < 2) day = '0' + day;
    return [year, month, day].join('-');
  }

  // Callback to close nationality picker if needed (can be empty)
  const onNationalityOpen = useCallback(() => {
    // Close other pickers if they existed
  }, []);

  // TODO: Add validation logic similar to SignUpScreen if needed
  const validateProfileForm = () => {
      // Add validation rules for fields being edited
      const newErrors = {};
      if (isEditing) { // Only validate if editing
          if (!firstName.trim()) newErrors.firstName = 'First name is required.';
          if (!lastName.trim()) newErrors.lastName = 'Last name is required.';
          if (birthDate && !/^\d{4}-\d{2}-\d{2}$/.test(birthDate)) newErrors.birthDate = 'Please use YYYY-MM-DD format.';
          if (!nationalityValue) newErrors.nationality = 'Nationality is required.'; // Validate picker value
          if (!whatsapp.trim()) newErrors.whatsapp = 'WhatsApp number is required.';
          else if (!/^\+\d{7,}$/.test(whatsapp.trim())) newErrors.whatsapp = 'Invalid WhatsApp format (e.g., +1234...).';
      }
      setErrors(newErrors);
      return Object.keys(newErrors).length === 0;
  }

  const handleSaveChanges = async () => {
    if (!user || !validateProfileForm()) {
        if (!validateProfileForm()) Alert.alert("Validation Error", "Please check the fields.");
        return;
    };
    setIsSaving(true);
    try {
        // Prepare only the fields that might have changed
        const profileData = {
            first_name: firstName.trim(),
            last_name: lastName.trim(),
            birth_date: birthDate || null, // Send null if empty, ensure backend handles it
            nationality: nationalityValue, // Send value from picker state
            whatsapp: whatsapp.trim(),
        };
        console.log("Saving profile data:", profileData);
        const response = await authService.updateProfile(profileData);
        if (response.status === 'success' && response.data) {
            Alert.alert("Success", "Profile updated successfully!");
            // Update user in AuthContext using the provided function
            updateUserState(response.data);
            setIsEditing(false); // Exit edit mode after successful save
        } else {
            throw new Error(response.message || "Failed to update profile");
        }
    } catch (error) {
        console.error("Error updating profile:", error);
        Alert.alert("Error", error.message || "Could not update profile.");
    } finally {
        setIsSaving(false);
    }
  };

  const handleCancelEdit = () => {
      // Reset fields to original user data
      if (user) {
        setFirstName(user.first_name || '');
        setLastName(user.last_name || '');
        setBirthDate(user.birth_date ? user.birth_date.split('T')[0] : '');
        setNationalityValue(user.nationality || null); // Reset picker value
        setWhatsapp(user.whatsapp || '');
      }
      setErrors({}); // Clear errors
      setIsEditing(false); // Exit edit mode
  };

  return (
    <ScrollView style={styles.container}>
      <Text style={styles.title}>My Profile</Text>
      {/* Display User Info */}
      {user ? (
        <View>
          {/* Use TextInput when editing, Text otherwise */}
          <TextInput
            label="First Name"
            value={firstName}
            onChangeText={setFirstName}
            editable={isEditing}
            error={errors.firstName}
            style={!isEditing ? styles.readOnlyInput : null}
          />
           <TextInput
            label="Last Name"
            value={lastName}
            onChangeText={setLastName}
            editable={isEditing}
            error={errors.lastName}
            style={!isEditing ? styles.readOnlyInput : null}
          />
          <TextInput
            label="Email"
            value={user.email} // Email is not editable in this flow
            editable={false}
            style={styles.readOnlyInput} // Make it look read-only
          />
           {/* Birth Date - Show Text or Picker */}
           <View style={styles.inputContainer}>
             <Text style={styles.label}>Birth Date</Text>
             {isEditing ? (
               <>
                 <TouchableOpacity onPress={() => setShowBirthDatePicker(true)} style={styles.dateDisplay}>
                   {/* Display formatted date string */}
                   <Text style={styles.dateText}>{birthDate || 'Select Date'}</Text>
                 </TouchableOpacity>
                 {showBirthDatePicker && (
                   <DateTimePicker
                     testID="birthDatePicker"
                     value={birthDate ? new Date(birthDate + 'T00:00:00') : new Date()} // Ensure valid Date object
                     mode="date"
                     display="default"
                     onChange={onBirthDateChange}
                     maximumDate={new Date(Date.now() - 18 * 365 * 24 * 60 * 60 * 1000)}
                   />
                 )}
               </>
             ) : (
               <Text style={[styles.value, styles.readOnlyValue]}>{birthDate ? new Date(birthDate + 'T00:00:00').toLocaleDateString() : 'N/A'}</Text>
             )}
             {isEditing && errors.birthDate && <Text style={styles.errorText}>{errors.birthDate}</Text>}
           </View>
           {/* Nationality - Show Text or Picker */}
            <View style={[styles.inputContainer, isEditing && { zIndex: 3000 }]}>
              <Text style={styles.label}>Nationality</Text>
              {isEditing ? (
                <DropDownPicker
                    open={nationalityOpen}
                    value={nationalityValue}
                    items={nationalityItems}
                    setOpen={setNationalityOpen}
                    setValue={setNationalityValue}
                    setItems={setNationalityItems}
                    onOpen={onNationalityOpen}
                    searchable={true}
                    placeholder="Select your nationality"
                    listMode="MODAL"
                    style={styles.dropdown}
                    dropDownContainerStyle={styles.dropdownContainer}
                    searchPlaceholder="Search..."
                    zIndex={3000}
                    zIndexInverse={1000}
                 />
              ) : (
                 <Text style={[styles.value, styles.readOnlyValue]}>{nationalityValue || 'N/A'}</Text>
              )}
               {isEditing && errors.nationality && <Text style={styles.errorText}>{errors.nationality}</Text>}
            </View>
           <TextInput
            label="WhatsApp"
            value={whatsapp}
            onChangeText={setWhatsapp}
            placeholder="+1234567890"
            keyboardType="phone-pad"
            editable={isEditing}
            error={errors.whatsapp}
            style={!isEditing ? styles.readOnlyInput : null}
          />
          {/* Edit/Save/Cancel Buttons */}
          <View style={styles.buttonContainer}>
            {isEditing ? (
              <>
                <Button
                    title="Save Changes"
                    onPress={handleSaveChanges}
                    loading={isSaving}
                    style={[styles.button, styles.saveButton]}
                />
                 <Button
                    title="Cancel"
                    onPress={handleCancelEdit}
                    style={[styles.button, styles.cancelButton]}
                    textStyle={styles.cancelButtonText}
                    disabled={isSaving}
                />
              </>
            ) : (
               <Button
                    title="Edit Profile"
                    onPress={() => setIsEditing(true)}
                    style={[styles.button, styles.editButton]}
                />
            )}
          </View>
       </View>
      ) : (
        <Text>Loading profile...</Text>
      )}
       {/* Add Logout Button Here (Alternative to header) */}
       <TouchableOpacity onPress={logout} style={styles.logoutButton}>
            <Text style={styles.logoutButtonText}>Log Out</Text>
       </TouchableOpacity>
    </ScrollView>
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
  label: {
      fontSize: 16,
      fontWeight: '500',
      color: '#555',
      marginTop: 15,
  },
  value: {
      fontSize: 16,
      color: '#333',
      marginBottom: 5,
  },
  readOnlyInput: { // Style to make TextInput look like Text
      backgroundColor: '#eee', // Light grey background
      color: '#555',
      borderWidth: 0,
      paddingVertical: 15, // Match picker display padding
      minHeight: 50, // Match picker display height
      paddingHorizontal: 0, // Remove horizontal padding for text alignment
  },
   readOnlyValue: { // Specific style for Text when read-only
      paddingVertical: 15, // Match picker display padding
      minHeight: 50, // Match picker display height
      textAlign: 'left', // Align text like input
  },
  buttonContainer: {
      marginTop: 30,
  },
  button: { // Common button style adjustments
      width: 'auto', // Allow buttons to size to content
      alignSelf: 'center',
      paddingHorizontal: 30,
      marginBottom: 10,
  },
  saveButton: {
      backgroundColor: '#007bff', // Blue color for save
  },
  editButton: {
       backgroundColor: '#6c757d', // Grey for edit
  },
  cancelButton: {
       backgroundColor: '#f8f9fa', // Light background for cancel
       borderColor: '#6c757d',
       borderWidth: 1,
  },
   cancelButtonText: {
       color: '#6c757d',
   },
   // Add styles from SignUpScreen for pickers
    dateDisplay: {
      borderWidth: 1,
      borderColor: '#ccc',
      borderRadius: 8,
      paddingHorizontal: 15,
      paddingVertical: 15,
      backgroundColor: '#fff',
      justifyContent: 'center',
      minHeight: 50,
  },
  dateText: {
      fontSize: 16,
      color: '#333',
  },
  dropdown: {
      borderColor: '#ccc',
      minHeight: 50,
  },
  dropdownContainer: {
      borderColor: '#ccc',
  },
   errorText: {
      marginTop: 4,
      color: 'red',
      fontSize: 12,
  },
  logoutButton: {
    marginTop: 15, // Reduced margin
    backgroundColor: '#dc3545', // Red color for logout
    paddingVertical: 12,
    paddingHorizontal: 25,
    borderRadius: 8,
    alignSelf: 'center', // Center the button
  },
  logoutButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: 'bold',
  },
});

export default ProfileScreen;