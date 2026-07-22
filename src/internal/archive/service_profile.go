// service_profile.go — user profile management: load, save, logout, list users.
package archive

import "errors"

func (s *Service) LoadProfile() (*Profile, error) {
	if s.store == nil {
		return &Profile{}, nil
	}
	return s.store.LoadProfile()
}

func (s *Service) SaveProfile(profile Profile) (*Profile, error) {
	if s.store == nil {
		return nil, errors.New("archive store is unavailable")
	}
	id, err := s.store.SaveProfile(profile)
	if err != nil {
		return nil, err
	}
	if profile.LoggedIn {
		s.SetActiveUser(id)
	}
	return s.store.GetProfileByID(id)
}

func (s *Service) Logout() (*Profile, error) {
	if s.store == nil {
		return nil, errors.New("archive store is unavailable")
	}
	if err := s.store.Logout(); err != nil {
		return nil, err
	}
	// Drop per-user cached platform state so the next login reloads cleanly.
	s.platforms = make(map[string]*PlatformState)
	return s.store.ActiveUser()
}

func (s *Service) ActiveUser() (*Profile, error) {
	if s.store == nil {
		return &Profile{}, nil
	}
	return s.store.ActiveUser()
}

func (s *Service) AvailableUsers() ([]Profile, error) {
	if s.store == nil {
		return []Profile{}, nil
	}
	return s.store.AvailableUsers()
}
