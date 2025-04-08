# 🚀Implémentation géolocalisation et cartographie open source dans RideShare

## 🎯 Objectif
Améliorer la page **"Create New Ride"** de l’application mobile **RideShare** (React Native + Expo) en :
- Ajoutant la **saisie intelligente** des lieux de départ et d’arrivée avec **autocomplétion**
- Affichant une **carte interactive** avec les points sélectionnés (via OpenStreetMap)
- Permettant de **visualiser la géolocalisation** sur une carte lors de la consultation d’un trajet
- Ajoutant une **demande d'autorisation de géolocalisation** lors de la première connexion utilisateur

---

## 🧱 Stack cible
- **Frontend** : React Native (avec Expo ou modules compatibles)
- **Cartographie** : Leaflet (via WebView ou composants React compatibles)
- **Géocodage + autocomplétion** : Photon API
- **Routage + itinéraire** : OpenRouteService API
- **Cartes** : OpenStreetMap
- **Backend** : Go (récupération des coordonnées si besoin)
- **BDD** : Supabase

---

## 🗺️ Comportement attendu

### 🔹 Page : `Create New Ride`
- Champs :
  - **Departure location** (champ texte avec autocomplétion via Photon)
  - **Arrival location** (champ texte avec autocomplétion via Photon)
  - **Date de départ** (calendrier)
  - **Heure de départ** (sélecteur d'heure)
  - **Nombre de places** (liste 1 à 5)

- En temps réel :
  - Les champs "Departure" et "Arrival" utilisent **Photon** pour afficher des suggestions
  - Une carte s’affiche avec **Leaflet** une fois les deux points définis
  - Utiliser **OpenRouteService** pour afficher une ligne de trajet sur la carte
  - Marqueur avec `popup` : "Voir sur Maps" (ouvre lien vers OpenStreetMap)

---

## 📍 Affichage de la carte interactive

### Technologie
- Utiliser **Leaflet** embarqué dans une **WebView React Native** (ou bibliothèque dédiée comme `react-native-maps` avec custom tiles OpenStreetMap)

### Comportement
- Une carte est affichée avec :
  - Marqueur pour le point de départ
  - Marqueur pour le point d’arrivée
  - Ligne d’itinéraire entre les deux (OpenRouteService)
  - Zoom automatique sur la zone englobant les deux points
  - Deux boutons sous la carte :
    - "📍 Voir départ sur Maps" → Ouvre OpenStreetMap avec le marqueur départ
    - "📍 Voir arrivée sur Maps" → Ouvre OpenStreetMap avec le marqueur arrivée

---

## 🧭 Géolocalisation du user au login

### Fonctionnalité
- Lors de la première connexion utilisateur :
  - Demander **l’autorisation d’accéder à la géolocalisation**
  - Si acceptée, récupérer les **coordonnées GPS actuelles**
  - Stocker dans Supabase (`users.location` si champ présent)
  - Permet d'afficher des trajets proches ou remplir automatiquement le champ "Departure" plus tard

### Technique
- Utiliser `expo-location` (ou `react-native-geolocation-service`)
- Gérer les permissions (Android + iOS)
- Fonction fallback si permission refusée

---

## 🧪 Cas d’usage testables

- L’utilisateur tape "Hoi" → autocomplétion affiche "Hoi An, Quang Nam, Vietnam"
- Après sélection, la carte affiche Hoi An
- Le user ajoute "Da Nang Airport" en arrivée → carte affiche le trajet
- Il clique sur "Voir arrivée sur Maps" → OpenStreetMap s’ouvre dans un navigateur
- Il se connecte pour la première fois → permission géolocalisation → position actuelle récupérée

---

## 🧩 Supabase – Structure à adapter

### `rides` table :
- `departure_location_name` (string)
- `departure_coords` (json ou float[2])
- `arrival_location_name` (string)
- `arrival_coords` (json ou float[2])

---

## 🌐 APIs open source à intégrer

### Photon API (autocomplétion)
- URL : `https://photon.komoot.io/api/?q=<QUERY>&lang=en`
- Réponse : liste de suggestions + coordonnées
- Gratuit et hébergé publiquement (option d’auto-hébergement possible)

### OpenRouteService API
- Route entre deux coordonnées GPS
- URL : `https://api.openrouteservice.org/v2/directions/driving-car`
- API key gratuite (jusqu’à 500 requêtes/jour en plan gratuit)
- Réponse : géojson à afficher sur la carte Leaflet

---

## 📌 Remarques techniques

- Utiliser OpenStreetMap respecte la licence ODbL (Open Database License)
- Afficher un texte en pied de carte : "© OpenStreetMap contributors"
- Prévoir un fallback (pas de carte) si aucun point n’est sélectionné
- Pour l’hébergement photon/ORS : garder l’usage public pour le MVP, passer à hébergement propre si la charge augmente

---

## ✅ Résultat attendu

> Une application RideShare dont la page de création de trajet est enrichie avec des suggestions d’adresse, une carte interactive montrant le trajet, et une expérience complète, sans dépendance à Google Maps.

