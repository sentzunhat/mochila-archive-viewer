// store_profiles.go — user profile CRUD: create, read, login/logout, list.
package archive

import (
	"database/sql"
	"errors"
)

// LoadProfile returns the first profile in the database (legacy single-user path).
// For multi-user flows, prefer ActiveUser or GetProfileByID.
func (s *Store) LoadProfile() (*Profile, error) {
	var profile Profile
	var loggedIn int
	err := s.db.QueryRow(`
		SELECT profile_id, username, full_name, logged_in
		FROM profile
		ORDER BY profile_id
		LIMIT 1
	`).Scan(&profile.ID, &profile.Username, &profile.FullName, &loggedIn)
	if errors.Is(err, sql.ErrNoRows) {
		return &Profile{}, nil
	}
	if err != nil {
		return nil, err
	}
	profile.LoggedIn = loggedIn == 1
	return &profile, nil
}

// GetProfileByID returns the profile with the given id, or an empty Profile when not found.
func (s *Store) GetProfileByID(id int64) (*Profile, error) {
	var profile Profile
	var loggedIn int
	err := s.db.QueryRow(`
		SELECT profile_id, username, full_name, logged_in
		FROM profile
		WHERE profile_id = ?
		LIMIT 1
	`, id).Scan(&profile.ID, &profile.Username, &profile.FullName, &loggedIn)
	if errors.Is(err, sql.ErrNoRows) {
		return &Profile{}, nil
	}
	if err != nil {
		return nil, err
	}
	profile.LoggedIn = loggedIn == 1
	return &profile, nil
}

// SaveProfile inserts or updates a profile and returns its id.
// Logging a profile in is exclusive: all other profiles are logged out first,
// and both operations run in a single transaction.
func (s *Store) SaveProfile(profile Profile) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if profile.LoggedIn {
		if _, err := tx.Exec("UPDATE profile SET logged_in = 0"); err != nil {
			return 0, err
		}
	}
	var existingID int64
	err = tx.QueryRow("SELECT profile_id FROM profile WHERE username = ?", profile.Username).Scan(&existingID)
	if errors.Is(err, sql.ErrNoRows) {
		result, err := tx.Exec(`
			INSERT INTO profile (username, full_name, logged_in)
			VALUES (?, ?, ?)
		`, profile.Username, profile.FullName, boolToInt(profile.LoggedIn))
		if err != nil {
			return 0, err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return 0, err
		}
		return id, tx.Commit()
	} else if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(`
		UPDATE profile SET full_name = ?, logged_in = ?, updated_at = CURRENT_TIMESTAMP
		WHERE profile_id = ?
	`, profile.FullName, boolToInt(profile.LoggedIn), existingID); err != nil {
		return 0, err
	}
	return existingID, tx.Commit()
}

// Logout sets all profiles to logged_in = 0.
func (s *Store) Logout() error {
	_, err := s.db.Exec("UPDATE profile SET logged_in = 0")
	return err
}

// ActiveUser returns the currently logged-in profile, or an empty Profile when none.
func (s *Store) ActiveUser() (*Profile, error) {
	var profile Profile
	var loggedIn int
	err := s.db.QueryRow("SELECT profile_id, username, full_name, logged_in FROM profile WHERE logged_in = 1 LIMIT 1").
		Scan(&profile.ID, &profile.Username, &profile.FullName, &loggedIn)
	if err != nil {
		return &Profile{}, nil
	}
	profile.LoggedIn = loggedIn == 1
	return &profile, nil
}

// AvailableUsers returns all known profiles ordered by username.
func (s *Store) AvailableUsers() ([]Profile, error) {
	rows, err := s.db.Query("SELECT profile_id, username, full_name, logged_in FROM profile ORDER BY username")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Profile
	for rows.Next() {
		var p Profile
		var loggedIn int
		if err := rows.Scan(&p.ID, &p.Username, &p.FullName, &loggedIn); err != nil {
			return nil, err
		}
		p.LoggedIn = loggedIn == 1
		result = append(result, p)
	}
	return result, rows.Err()
}
