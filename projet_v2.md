# 🛠️ Application RideShare (Amélioration MVP)

## 🎯 Objectif
Créer une application mobile avec React Native + Expo, backend en Go, base de données Supabase, et intégration Stripe (sandbox pour le moment).  
L'application RideShare permet à des **voyageurs** de partager leurs trajets au Vietnam et en Asie du Sud-Est.

---

## 🧾 Formulaire d’inscription (Signup Form)

Champs requis :
- **Prénom** (text)
- **Nom** (text)
- **Email** (text)
- **Mot de passe** (text, masqué)
- **Date de naissance** :
  - 📅 Sélecteur de **calendrier** (pas champ texte)
- **Nationalité** :
  - 📋 Liste déroulante **filtrable** par texte
  - Source : liste complète des nationalités ISO
- **Numéro WhatsApp** :
  - Préfixe pays : menu déroulant avec indicatifs (ex: `+33`, `+84`, etc.)
  - Numéro local : champ texte

Les données doivent être envoyées à Supabase, avec validation des champs côté client (React Native).

---

## 📱 Navigation globale

Barre de navigation visible sur toutes les pages (`BottomTabNavigator` avec React Navigation) :

- `Search a ride` (page par défaut)
- `My rides`
- `Profile`
- `Settings`

---

## 🔍 Search a Ride (Page principale)

Fonction : rechercher un trajet parmi les trajets disponibles.

Champs :
- **Lieu de départ** : champ texte
- **Lieu d’arrivée** : champ texte
- **Date** : sélection via **calendrier**

Comportement :
- Les champs sont **optionnels**
- Si aucun champ n’est rempli, et que l’utilisateur clique sur "Rechercher", il est redirigé vers **Available Rides**
- Sinon, les résultats sont filtrés selon les critères
- Les résultats sont redirigés vers la page **Available Rides**

Filtrage automatique :
- Les rides expirés (heure + date < now) ou complets (tous les sièges pris) **ne doivent pas s’afficher**
- Ces rides doivent être déplacés dans une table logique :  
  `historic_rides` ou `rides` avec un champ `status: "active" | "archived"`

---

## 📄 Available Rides Page

Affiche les résultats issus de la recherche.  
Chaque ride affiché contient :
- Lieu de départ
- Lieu d’arrivée
- Date & heure
- Nombre de places restantes
- Bouton "Rejoindre ce ride"

Comportement :
- En cliquant sur rejoindre :
  - Paiement via Stripe (sandbox)
  - Webhook côté backend confirme le paiement
  - Mise en relation par affichage des **numéros WhatsApp** (user + créateur)

---

## 🛣️ Create New Ride Page

Formulaire pour créer un trajet.

Champs :
- **Lieu de départ** (champ texte)
- **Lieu d’arrivée** (champ texte)
- **Date** : calendrier
- **Heure** : sélecteur d’heure (pas texte libre)
- **Nombre de places disponibles** : menu de 1 à 5

Données stockées dans Supabase, dans la table `rides`, associées au `user_id`.

---

## 📦 My Rides Page

Deux sections :
- **Mes rides créés** :
  - Liste des rides où `user_id = current_user`
  - Bouton **Supprimer ride** :
    - Si aucun participant : suppression directe
    - Si au moins un participant : pop-up ⚠️  
      > "Attention : un ou plusieurs participants ont déjà payé. Aucun remboursement ne sera effectué."

- **Mes rides en tant que participant** :
  - Bouton **Quitter ride**
    - Avertissement : "Vous ne serez pas remboursé en quittant ce trajet."

- **Historique** : rides passés → même logique que la page principale.

---

## 🙍 Profile Page

Affiche les infos de l'utilisateur :
- Nom, prénom
- Email
- Date de naissance
- Nationalité
- WhatsApp

Chaque champ peut être **modifié** (avec validation), et mis à jour dans Supabase.

---

## ⚙️ Settings Page

Contient :
- **Paramètres de paiement** (Stripe - à venir)
- **Notifications** (optionnel)
- **Suppression de compte** :
  - Supprime toutes les données liées à l’utilisateur dans Supabase
  - ✅ Inclure confirmation par alerte de sécurité ("Supprimer définitivement mon compte")

---

## 🧩 Backend (Go)

Endpoints requis :
- `POST /signup` → envoie à Supabase
- `POST /login`
- `POST /rides/create`
- `GET /rides/search`
- `GET /rides/available`
- `DELETE /rides/:id`
- `POST /rides/join`
- `POST /payment/create-stripe-session`
- `POST /webhook/stripe` → traitement post-paiement

---

## 🗃️ Supabase – Tables proposées

### `users`
- id (uuid)
- email
- password (hash)
- nom
- prenom
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
- nb_places_prises
- status (`active`, `archived`)

### `participants`
- id
- ride_id
- user_id
- status (`pending`, `paid`, `cancelled`)

### `payments`
- id
- stripe_session_id
- user_id
- ride_id
- status (`paid`, `failed`)
- date_paiement

---

## 🧪 Tests recommandés

- Tester l’inscription avec tous les types de données (dates, listes déroulantes)
- Tester la création et la recherche de ride
- Vérifier la non-affichage des rides expirés ou complets
- Simuler un paiement Stripe → vérifier webhook, stockage transaction, et affichage WhatsApp

---
