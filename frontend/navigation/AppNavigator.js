// frontend/navigation/AppNavigator.js
import React from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createStackNavigator } from '@react-navigation/stack';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs'; // Import Bottom Tabs
import { View, Text, StyleSheet, ActivityIndicator } from 'react-native'; // Added ActivityIndicator
import Ionicons from '@expo/vector-icons/Ionicons'; // Import icons

// Import Screens
import LoginScreen from '../screens/LoginScreen';
import SignUpScreen from '../screens/SignUpScreen';
import RidesListScreen from '../screens/RidesListScreen';
import CreateRideScreen from '../screens/CreateRideScreen';
import RideDetailScreen from '../screens/RideDetailScreen';
// Import the new tab screens
import SearchRideScreen from '../screens/SearchRideScreen';
import MyRidesScreen from '../screens/MyRidesScreen';
import ProfileScreen from '../screens/ProfileScreen';
import SettingsScreen from '../screens/SettingsScreen';
// Import Auth Context hook
import { useAuth } from '../contexts/AuthContext';
// Import TouchableOpacity for logout button (if used)
// import { TouchableOpacity } from 'react-native';

// Remove placeholder screen definitions as we now import the actual screens

// Placeholder component for the logout button (can be removed if not needed in header)
// function LogoutButton() { ... } // Keep if needed, or remove if logout is elsewhere

// Placeholder LoadingScreen component
function LoadingScreen() {
  return (
    <View style={styles.loadingContainer}>
      <ActivityIndicator size="large" color="#007bff" />
      <Text>Loading...</Text>
    </View>
  );
}

const AuthStack = createStackNavigator();
const Tab = createBottomTabNavigator(); // Create Tab Navigator instance
const HomeStack = createStackNavigator(); // Stack for screens reachable from tabs (e.g., Ride Details)

// Authentication Stack Navigator (Login, SignUp)
const AuthStackNavigator = () => (
  <AuthStack.Navigator screenOptions={{ headerShown: false }}>
    <AuthStack.Screen name="Login" component={LoginScreen} />
    <AuthStack.Screen name="SignUp" component={SignUpScreen} />
  </AuthStack.Navigator>
);

// Stack Navigator for the "Search" Tab flow
const SearchStackNavigator = () => (
  <HomeStack.Navigator // Reusing HomeStack instance, or create SearchStack = createStackNavigator();
     screenOptions={{
        headerStyle: { backgroundColor: '#007bff' },
        headerTintColor: '#fff',
        headerTitleStyle: { fontWeight: 'bold' },
    }}
  >
    <HomeStack.Screen name="SearchRide" component={SearchRideScreen} options={{ title: 'Search a Ride' }}/>
    {/* AvailableRidesScreen will display results from SearchRideScreen or all rides */}
    <HomeStack.Screen name="AvailableRides" component={RidesListScreen} options={{ title: 'Available Rides' }}/>
    <HomeStack.Screen name="CreateRide" component={CreateRideScreen} options={{ title: 'Create New Ride' }}/>
    <HomeStack.Screen name="RideDetail" component={RideDetailScreen} options={{ title: 'Ride Details' }}/>
  </HomeStack.Navigator>
);

// We might need similar Stacks for other tabs if they navigate to sub-screens
// For now, let's assume MyRides, Profile, Settings are single screens within the tab.


// Main Tab Navigator
const MainTabNavigator = () => (
  <Tab.Navigator
    screenOptions={({ route }) => ({
      tabBarIcon: ({ focused, color, size }) => {
        let iconName;
        // Assign icons based on route name
        if (route.name === 'Search') {
          iconName = focused ? 'search' : 'search-outline';
        } else if (route.name === 'MyRides') {
          iconName = focused ? 'list' : 'list-outline'; // Changed from car
        } else if (route.name === 'Profile') {
          iconName = focused ? 'person' : 'person-outline';
        } else if (route.name === 'Settings') {
          iconName = focused ? 'settings' : 'settings-outline';
        }
        // You can return any component that you like here!
        return <Ionicons name={iconName} size={size} color={color} />;
      },
      tabBarActiveTintColor: '#007bff', // Color for active tab
      tabBarInactiveTintColor: 'gray',   // Color for inactive tabs
      headerShown: false, // Hide default header for tabs, use Stack header instead
    })}
  >
    {/* Define the Tabs */}
    {/* Use HomeStackNavigator for the Search tab to allow navigation within it */}
    {/* Use SearchStackNavigator for the Search tab */}
    <Tab.Screen name="Search" component={SearchStackNavigator} options={{ title: 'Search' }}/>
    <Tab.Screen name="MyRides" component={MyRidesScreen} options={{ title: 'My Rides' }}/>
    <Tab.Screen name="Profile" component={ProfileScreen} options={{ title: 'Profile' }}/>
    <Tab.Screen name="Settings" component={SettingsScreen} options={{ title: 'Settings' }}/>
  </Tab.Navigator>
); // Added missing closing parenthesis and semicolon


// Main application navigator - decides which stack to show
export default function AppNavigator() {
  const { token, isLoading } = useAuth(); // Get token and loading state from context

  // Show loading indicator while checking auth state
  if (isLoading) {
    return <LoadingScreen />;
  }

  // Render the correct navigator based on token presence
  return (
    <NavigationContainer>
      {token ? <MainTabNavigator /> : <AuthStackNavigator />}
    </NavigationContainer>
  );
}

// Styles
const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    padding: 20,
  },
  loadingContainer: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
  },
  welcomeText: {
    fontSize: 22,
    fontWeight: 'bold',
    marginBottom: 20,
  },
  // Removed headerButton styles as logout is moved or handled differently
});