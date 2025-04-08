# 🚘 RideShare - Présentation du MVP

## 🏷️ Nom du produit
**RideShare**

## 🧩 Type de produit
Application mobile (iOS & Android) basée sur le principe du covoiturage entre voyageurs utilisant des trajets de taxis privés (comme Grab) au Vietnam et en Asie du Sud-Est.

## 🎯 Objectif
Offrir une solution simple et économique pour les voyageurs étrangers en difficulté de transport en leur permettant de partager des trajets de type Grab avec d'autres personnes allant dans la même direction.  
L’application met en relation les utilisateurs, puis se désengage une fois le contact établi, avec un frais fixe de **2 €** par utilisateur (non remboursable).

## 🧑‍🤝‍🧑 Cible
- Touristes internationaux (principalement européens et américains)
- Routards, voyageurs solo, backpackers
- Digital nomads
- Groupes de touristes souhaitant optimiser leurs frais de transport

## 🌍 Zone géographique ciblée
- **Vietnam** (zone principale de lancement)
  - Focus initial : Danang ↔ Hoi An
- **Asie du Sud-Est** à moyen terme (Thaïlande, Cambodge, Laos, Indonésie)

---

# 🛠️ Prompt technique d’implémentation du MVP

## Stack technique

- **Frontend** : React Native + Expo
- **Backend** : Go (avec Fiber framework)
- **Base de données** : Supabase
- **Paiement** : Stripe (d'abord en mode développeur / sandbox)

## Modules fonctionnels

### Authentification
- Email + mot de passe
- Enregistrement du numéro WhatsApp (utilisé pour la mise en relation)
- Champs utilisateurs :
  - Nom, Prénom
  - Date de naissance
  - Nationalité
  - Email
  - Mot de passe (hashé)
  - Numéro WhatsApp

### Gestion des trajets
- Création d’un trajet :
  - Départ
  - Arrivée
  - Date et heure
  - Nombre de sièges disponibles (1 à 5)
- Affichage des trajets publics
- Filtrage par lieu et date
- Rejoindre un trajet

### Paiement (Stripe)
- Paiement de 2 € déclenché :
  - Quand un utilisateur rejoint un trajet
  - Quand quelqu’un rejoint un trajet publié par l’utilisateur
- Mode **sandbox Stripe** pour tests initiaux
- Déclenchement du paiement via l’API backend
- Déclenchement d’un webhook Stripe : 
  - Si paiement réussi ⇒ envoi des numéros WhatsApp aux deux parties

## Structure des tables Supabase

### `users`
- id
- email
- password (hash)
- nom, prénom
- date_naissance
- nationalité
- whatsapp

### `rides`
- id
- user_id (créateur)
- lieu_depart
- lieu_arrivee
- date
- heure
- nb_places_dispo
- statut (ouvert / fermé)

### `participants`
- id
- user_id
- ride_id
- statut (en attente / confirmé)

### `transactions`
- id
- user_id
- ride_id
- stripe_payment_id
- statut (en attente / payé)

---

## Étapes d’implémentation technique (Roadmap Dev)

### 1. Setup
- Initialiser projet Go backend avec Fiber
- Initialiser projet React Native avec Expo
- Créer le projet Supabase + tables

### 2. Authentification
- Formulaire d'inscription + login frontend
- API Go : `/signup`, `/login`
- Connexion à Supabase pour gestion des utilisateurs

### 3. Gestion des trajets
- Création de trajets (formulaire + API)
- Listing des trajets disponibles
- Fonction de participation à un trajet

### 4. Paiement Stripe
- Intégration Stripe sandbox (API Go)
- Génération de lien de paiement ou Stripe PaymentIntent
- Gestion des webhooks Stripe pour valider les paiements

### 5. Déblocage WhatsApp
- Une fois paiement validé → afficher numéros WhatsApp entre participants

### 6. Tests finaux
- Tests des différents cas d’usage
- Tests des paiements Stripe en sandbox
- Affinage UI/UX mobile

---

